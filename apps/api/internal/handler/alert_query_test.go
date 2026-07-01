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
