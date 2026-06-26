package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRequireMutateAllowsMember(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SetUserRole("member"), RequireMutate())
	r.POST("/x", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/x", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)
}

func TestRequireMutateBlocksViewer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SetUserRole("viewer"), RequireMutate())
	r.POST("/x", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/x", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusForbidden, w.Code)
}
