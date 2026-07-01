package db

import (
	"context"
)

const DefaultAlertListLimit = 100

type ListAlertsParams struct {
	Query  string
	Limit  int
	Offset int
}

func (s *Store) ListAlerts(ctx context.Context, params ListAlertsParams) ([]Alert, error) {
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

	const q = `
SELECT id, fingerprint, status, severity, title, body, labels, raw_payload, received_at
FROM alerts
WHERE ($1 = '' OR search_tsv @@ websearch_to_tsquery('english', $1))
ORDER BY received_at DESC
LIMIT $2 OFFSET $3`

	rows, err := s.pool.Query(ctx, q, params.Query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	alerts := make([]Alert, 0, limit)
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
