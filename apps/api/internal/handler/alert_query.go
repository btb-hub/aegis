package handler

import (
	"strconv"
	"strings"
	"time"

	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/aegis/aegis/pkg/db"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type parsedListAlertsQuery struct {
	Params   db.ListAlertsParams
	TeamID   *uuid.UUID
	Page     int
	PageSize int
}

func parseListAlertsQuery(c *gin.Context) (parsedListAlertsQuery, error) {
	page := 1
	if raw := strings.TrimSpace(c.Query("page")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 {
			return parsedListAlertsQuery{}, apperrors.Validation("page must be a positive integer", nil)
		}
		page = value
	}

	pageSize := db.DefaultAlertListLimit
	if raw := strings.TrimSpace(c.Query("page_size")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 {
			return parsedListAlertsQuery{}, apperrors.Validation("page_size must be a positive integer", nil)
		}
		if value > db.DefaultAlertListLimit {
			value = db.DefaultAlertListLimit
		}
		pageSize = value
	}

	params := db.ListAlertsParams{
		Query:        strings.TrimSpace(c.Query("q")),
		Severity:     strings.TrimSpace(c.Query("severity")),
		Status:       strings.TrimSpace(c.Query("status")),
		LabelFilters: map[string]string{},
		Limit:        pageSize,
		Offset:       (page - 1) * pageSize,
	}

	if fromRaw := strings.TrimSpace(c.Query("from")); fromRaw != "" {
		from, err := time.Parse(time.RFC3339, fromRaw)
		if err != nil {
			return parsedListAlertsQuery{}, apperrors.Validation("from must be RFC3339", nil)
		}
		params.From = &from
	}
	if toRaw := strings.TrimSpace(c.Query("to")); toRaw != "" {
		to, err := time.Parse(time.RFC3339, toRaw)
		if err != nil {
			return parsedListAlertsQuery{}, apperrors.Validation("to must be RFC3339", nil)
		}
		params.To = &to
	}

	for _, raw := range c.QueryArray("label") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		key, value, ok := strings.Cut(raw, ":")
		if !ok || strings.TrimSpace(key) == "" {
			return parsedListAlertsQuery{}, apperrors.Validation("label must be key:value", map[string]any{"label": raw})
		}
		params.LabelFilters[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}

	var teamID *uuid.UUID
	if raw := strings.TrimSpace(c.Query("team_id")); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			return parsedListAlertsQuery{}, apperrors.Validation("team_id must be a valid uuid", nil)
		}
		teamID = &id
	}

	return parsedListAlertsQuery{
		Params:   params,
		TeamID:   teamID,
		Page:     page,
		PageSize: pageSize,
	}, nil
}
