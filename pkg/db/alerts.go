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
