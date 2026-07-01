package db

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const DefaultAlertListLimit = 100

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
       id, fingerprint, status, severity, title, body, labels, raw_payload, received_at
%[2]s
ORDER BY %[1]s, received_at DESC`, groupExpr, from.sql)
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
			&body, &alert.Labels, &alert.RawPayload, &alert.ReceivedAt,
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
	params = normalizeListAlertsParams(params)
	q := buildAlertListQuery(params, `
SELECT id, fingerprint, status, severity, title, body, labels, raw_payload, received_at`)
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
		if err := rows.Scan(
			&alert.ID, &alert.Fingerprint, &alert.Status, &alert.Severity, &alert.Title,
			&body, &alert.Labels, &alert.RawPayload, &alert.ReceivedAt,
		); err != nil {
			return nil, err
		}
		alert.Body = body
		alerts = append(alerts, alert)
	}
	return alerts, rows.Err()
}
