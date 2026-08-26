package db

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

const DefaultAlertListLimit = 100

const alertListBaseSelect = `id, fingerprint, status, severity, title, body, labels, raw_payload, received_at`

const alertOpenIncidentIDSelect = `(
  SELECT ia.incident_id
  FROM incident_alerts ia
  JOIN incidents i ON i.id = ia.incident_id
  WHERE ia.alert_id = alerts.id
    AND i.status IN ('open', 'acknowledged')
  ORDER BY ia.created_at DESC
  LIMIT 1
) AS incident_id`

type ListAlertsParams struct {
	Query        string
	Severity     string
	Status       string
	LabelFilters map[string]string
	From         *time.Time
	To           *time.Time
	Limit        int
	Offset       int
}

type alertListQuery struct {
	sql  string
	args []any
}

func buildAlertListQuery(params ListAlertsParams, selectClause string) alertListQuery {
	var (
		conditions []string
		args       []any
		argPos     = 1
	)

	if params.Query != "" {
		conditions = append(conditions, fmt.Sprintf("search_tsv @@ websearch_to_tsquery('english', $%d)", argPos))
		args = append(args, params.Query)
		argPos++
	}
	if params.Severity != "" {
		conditions = append(conditions, fmt.Sprintf("severity = $%d", argPos))
		args = append(args, params.Severity)
		argPos++
	}
	if params.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argPos))
		args = append(args, params.Status)
		argPos++
	}
	if len(params.LabelFilters) > 0 {
		labelJSON, err := json.Marshal(params.LabelFilters)
		if err == nil {
			conditions = append(conditions, fmt.Sprintf("labels @> $%d::jsonb", argPos))
			args = append(args, labelJSON)
			argPos++
		}
	}
	if params.From != nil {
		conditions = append(conditions, fmt.Sprintf("received_at >= $%d", argPos))
		args = append(args, *params.From)
		argPos++
	}
	if params.To != nil {
		conditions = append(conditions, fmt.Sprintf("received_at <= $%d", argPos))
		args = append(args, *params.To)
		argPos++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	return alertListQuery{
		sql:  selectClause + " FROM alerts " + where,
		args: args,
	}
}

func normalizeListAlertsParams(params ListAlertsParams) ListAlertsParams {
	limit := params.Limit
	if limit <= 0 {
		limit = DefaultAlertListLimit
	}
	if limit > DefaultAlertListLimit {
		limit = DefaultAlertListLimit
	}
	offset := params.Offset
	if offset < 0 {
		offset = 0
	}
	params.Limit = limit
	params.Offset = offset
	if params.LabelFilters == nil {
		params.LabelFilters = map[string]string{}
	}
	return params
}

func alertListFromClause(params ListAlertsParams) alertListQuery {
	q := buildAlertListQuery(params, "SELECT 1")
	q.sql = strings.TrimPrefix(q.sql, "SELECT 1 ")
	return q
}

func appendAlertListCondition(from alertListQuery, condition string) alertListQuery {
	if strings.Contains(from.sql, " WHERE ") {
		return alertListQuery{sql: from.sql + " AND " + condition, args: from.args}
	}
	return alertListQuery{sql: from.sql + " WHERE " + condition, args: from.args}
}

type AlertGroupBy struct {
	Severity bool
	LabelKey string
}

type AlertGroupBucket struct {
	Key    string
	Count  int
	Sample *Alert
}

func groupByExpression(params AlertGroupBy, args []any) (expr string, newArgs []any) {
	newArgs = args
	if params.Severity {
		return "severity", newArgs
	}
	pos := len(newArgs) + 1
	newArgs = append(newArgs, params.LabelKey)
	return fmt.Sprintf("COALESCE(labels->>$%d, '')", pos), newArgs
}

func (s *Store) GroupAlerts(ctx context.Context, filters ListAlertsParams, groupBy AlertGroupBy) ([]AlertGroupBucket, error) {
	if groupBy.LabelKey == "" && !groupBy.Severity {
		return nil, fmt.Errorf("group by requires severity or label key")
	}
	filters = normalizeListAlertsParams(filters)
	from := alertListFromClause(filters)

	groupExpr, args := groupByExpression(groupBy, from.args)

	countSQL := fmt.Sprintf(
		"SELECT %s AS bucket_key, COUNT(*)::int AS bucket_count %s GROUP BY bucket_key ORDER BY bucket_count DESC",
		groupExpr, from.sql,
	)
	rows, err := s.pool.Query(ctx, countSQL, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	buckets := make([]AlertGroupBucket, 0)
	for rows.Next() {
		var bucket AlertGroupBucket
		if err := rows.Scan(&bucket.Key, &bucket.Count); err != nil {
			return nil, err
		}
		buckets = append(buckets, bucket)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(buckets) == 0 {
		return buckets, nil
	}

	sampleSQL := fmt.Sprintf(`
SELECT DISTINCT ON (%[1]s) %[1]s AS bucket_key,
       id, fingerprint, status, severity, title, body, labels, raw_payload, received_at,
       %[3]s
%[2]s
ORDER BY %[1]s, received_at DESC`, groupExpr, from.sql, alertOpenIncidentIDSelect)
	sampleRows, err := s.pool.Query(ctx, sampleSQL, args...)
	if err != nil {
		return nil, err
	}
	defer sampleRows.Close()

	samplesByKey := map[string]Alert{}
	for sampleRows.Next() {
		var key string
		var alert Alert
		var body *string
		if err := sampleRows.Scan(
			&key, &alert.ID, &alert.Fingerprint, &alert.Status, &alert.Severity, &alert.Title,
			&body, &alert.Labels, &alert.RawPayload, &alert.ReceivedAt, &alert.IncidentID,
		); err != nil {
			return nil, err
		}
		alert.Body = body
		samplesByKey[key] = alert
	}
	if err := sampleRows.Err(); err != nil {
		return nil, err
	}

	for i := range buckets {
		if sample, ok := samplesByKey[buckets[i].Key]; ok {
			sampleCopy := sample
			buckets[i].Sample = &sampleCopy
		}
	}
	return buckets, nil
}

func (s *Store) CountAlerts(ctx context.Context, params ListAlertsParams) (int, error) {
	params = normalizeListAlertsParams(params)
	q := buildAlertListQuery(params, "SELECT COUNT(*)")

	var total int
	if err := s.pool.QueryRow(ctx, q.sql, q.args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func (s *Store) ListAlerts(ctx context.Context, params ListAlertsParams) ([]Alert, error) {
	return s.queryAlerts(ctx, params, true)
}

func (s *Store) queryAlerts(ctx context.Context, params ListAlertsParams, includeIncidentID bool) ([]Alert, error) {
	params = normalizeListAlertsParams(params)
	selectClause := "SELECT " + alertListBaseSelect
	if includeIncidentID {
		selectClause += ", " + alertOpenIncidentIDSelect
	}
	q := buildAlertListQuery(params, selectClause)
	q.sql += fmt.Sprintf(" ORDER BY received_at DESC LIMIT $%d OFFSET $%d", len(q.args)+1, len(q.args)+2)
	q.args = append(q.args, params.Limit, params.Offset)

	rows, err := s.pool.Query(ctx, q.sql, q.args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	alerts := make([]Alert, 0, params.Limit)
	for rows.Next() {
		var alert Alert
		var body *string
		if includeIncidentID {
			if err := rows.Scan(
				&alert.ID, &alert.Fingerprint, &alert.Status, &alert.Severity, &alert.Title,
				&body, &alert.Labels, &alert.RawPayload, &alert.ReceivedAt, &alert.IncidentID,
			); err != nil {
				return nil, err
			}
		} else {
			if err := rows.Scan(
				&alert.ID, &alert.Fingerprint, &alert.Status, &alert.Severity, &alert.Title,
				&body, &alert.Labels, &alert.RawPayload, &alert.ReceivedAt,
			); err != nil {
				return nil, err
			}
		}
		alert.Body = body
		alerts = append(alerts, alert)
	}
	return alerts, rows.Err()
}

const alertExportBatchSize = 500

type AlertAnalytics struct {
	BySeverity map[string]int
	ByStatus   map[string]int
	TopLabels  []LabelCount
}

type LabelCount struct {
	Key   string
	Value string
	Count int
}

func (s *Store) AlertAnalytics(ctx context.Context, params ListAlertsParams, labelKey string) (AlertAnalytics, error) {
	from := alertListFromClause(params)

	severitySQL := fmt.Sprintf(
		"SELECT severity, COUNT(*)::int %s GROUP BY severity ORDER BY COUNT(*) DESC",
		from.sql,
	)
	severityRows, err := s.pool.Query(ctx, severitySQL, from.args...)
	if err != nil {
		return AlertAnalytics{}, err
	}
	defer severityRows.Close()

	bySeverity := map[string]int{}
	for severityRows.Next() {
		var key string
		var count int
		if err := severityRows.Scan(&key, &count); err != nil {
			return AlertAnalytics{}, err
		}
		bySeverity[key] = count
	}
	if err := severityRows.Err(); err != nil {
		return AlertAnalytics{}, err
	}

	statusSQL := fmt.Sprintf(
		"SELECT status, COUNT(*)::int %s GROUP BY status",
		from.sql,
	)
	statusRows, err := s.pool.Query(ctx, statusSQL, from.args...)
	if err != nil {
		return AlertAnalytics{}, err
	}
	defer statusRows.Close()

	byStatus := map[string]int{}
	for statusRows.Next() {
		var key string
		var count int
		if err := statusRows.Scan(&key, &count); err != nil {
			return AlertAnalytics{}, err
		}
		byStatus[key] = count
	}
	if err := statusRows.Err(); err != nil {
		return AlertAnalytics{}, err
	}

	topLabels := []LabelCount{}
	if labelKey != "" {
		labelPos := len(from.args) + 1
		labelFrom := appendAlertListCondition(from, fmt.Sprintf("labels ? $%d", labelPos))
		args := append(append([]any{}, labelFrom.args...), labelKey)
		labelSQL := fmt.Sprintf(`
SELECT $%d::text AS label_key, COALESCE(labels->>$%d, '') AS label_value, COUNT(*)::int AS label_count
%s
GROUP BY label_value
ORDER BY label_count DESC
LIMIT 10`, labelPos, labelPos, labelFrom.sql)
		labelRows, err := s.pool.Query(ctx, labelSQL, args...)
		if err != nil {
			return AlertAnalytics{}, err
		}
		defer labelRows.Close()

		for labelRows.Next() {
			var item LabelCount
			if err := labelRows.Scan(&item.Key, &item.Value, &item.Count); err != nil {
				return AlertAnalytics{}, err
			}
			topLabels = append(topLabels, item)
		}
		if err := labelRows.Err(); err != nil {
			return AlertAnalytics{}, err
		}
	}

	return AlertAnalytics{
		BySeverity: bySeverity,
		ByStatus:   byStatus,
		TopLabels:  topLabels,
	}, nil
}

func (s *Store) StreamAlertsCSV(ctx context.Context, params ListAlertsParams, w io.Writer) error {
	writer := csv.NewWriter(w)
	if err := writer.Write([]string{"id", "fingerprint", "status", "severity", "title", "body", "labels", "received_at"}); err != nil {
		return err
	}

	params = normalizeListAlertsParams(params)
	params.Limit = alertExportBatchSize
	params.Offset = 0

	for {
		alerts, err := s.queryAlerts(ctx, params, false)
		if err != nil {
			return err
		}
		if len(alerts) == 0 {
			break
		}
		for _, alert := range alerts {
			var labels map[string]string
			_ = json.Unmarshal(alert.Labels, &labels)
			labelsJSON, _ := json.Marshal(labels)
			body := ""
			if alert.Body != nil {
				body = *alert.Body
			}
			if err := writer.Write([]string{
				alert.ID.String(),
				alert.Fingerprint,
				alert.Status,
				alert.Severity,
				alert.Title,
				body,
				string(labelsJSON),
				alert.ReceivedAt.Format(time.RFC3339),
			}); err != nil {
				return err
			}
		}
		writer.Flush()
		if err := writer.Error(); err != nil {
			return err
		}
		if len(alerts) < alertExportBatchSize {
			break
		}
		params.Offset += alertExportBatchSize
	}
	writer.Flush()
	return writer.Error()
}
