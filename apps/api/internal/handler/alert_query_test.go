package handler

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestParseListAlertsQueryDefaults(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/api/v1/alerts", nil)

	parsed, err := parseListAlertsQuery(c)
	require.NoError(t, err)
	require.Equal(t, 1, parsed.Page)
	require.Equal(t, 100, parsed.PageSize)
	require.Equal(t, 100, parsed.Params.Limit)
	require.Equal(t, 0, parsed.Params.Offset)
}

func TestParseListAlertsQueryLabelFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/api/v1/alerts?label=env:prod&label=region:eu", nil)

	parsed, err := parseListAlertsQuery(c)
	require.NoError(t, err)
	require.Equal(t, "prod", parsed.Params.LabelFilters["env"])
	require.Equal(t, "eu", parsed.Params.LabelFilters["region"])
}

func TestParseListAlertsQueryInvalidLabel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/api/v1/alerts?label=invalid", nil)

	_, err := parseListAlertsQuery(c)
	require.Error(t, err)
}

func TestParseListAlertsQueryTimeRange(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(
		"GET",
		"/api/v1/alerts?from=2026-06-01T00:00:00Z&to=2026-06-30T23:59:59Z&severity=critical&status=firing&q=cpu&page=2&page_size=150",
		nil,
	)

	parsed, err := parseListAlertsQuery(c)
	require.NoError(t, err)
	require.Equal(t, "critical", parsed.Params.Severity)
	require.Equal(t, "firing", parsed.Params.Status)
	require.Equal(t, "cpu", parsed.Params.Query)
	require.Equal(t, 2, parsed.Page)
	require.Equal(t, 100, parsed.PageSize)
	require.Equal(t, 100, parsed.Params.Offset)
	require.NotNil(t, parsed.Params.From)
	require.NotNil(t, parsed.Params.To)
}

func TestParseListAlertsQueryInvalidFrom(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/api/v1/alerts?from=not-a-date", nil)

	_, err := parseListAlertsQuery(c)
	require.Error(t, err)
}

func TestParseListAlertsQueryInvalidTo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/api/v1/alerts?to=not-a-date", nil)

	_, err := parseListAlertsQuery(c)
	require.Error(t, err)
}

func TestParseListAlertsQueryInvalidPage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/api/v1/alerts?page=0", nil)

	_, err := parseListAlertsQuery(c)
	require.Error(t, err)
}

func TestParseListAlertsQueryInvalidTeamID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/api/v1/alerts?team_id=not-a-uuid", nil)

	_, err := parseListAlertsQuery(c)
	require.Error(t, err)
}

func TestParseListAlertsQuerySkipsEmptyLabel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/api/v1/alerts?label=&label=env:prod", nil)

	parsed, err := parseListAlertsQuery(c)
	require.NoError(t, err)
	require.Equal(t, "prod", parsed.Params.LabelFilters["env"])
	require.Len(t, parsed.Params.LabelFilters, 1)
}
