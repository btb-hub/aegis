package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRequireSameOrigin(t *testing.T) {
	for _, test := range []struct {
		name, origin, referer, fetchSite string
		allowed                          bool
	}{
		{name: "same origin", origin: "https://aegis.test", allowed: true},
		{name: "referer fallback", referer: "https://aegis.test/account", allowed: true},
		{name: "missing"},
		{name: "null", origin: "null"},
		{name: "foreign origin", origin: "https://evil.test"},
		{name: "foreign referer", referer: "https://evil.test/account"},
		{name: "wrong scheme", origin: "http://aegis.test"},
		{name: "wrong port", origin: "https://aegis.test:1234"},
		{name: "fetch cross site", origin: "https://aegis.test", fetchSite: "cross-site"},
		{name: "origin takes precedence", origin: "https://evil.test", referer: "https://aegis.test/account"},
		{name: "credentials", origin: "https://user:password@aegis.test"},
	} {
		t.Run(test.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()
			router.POST("/mutate", RequireSameOrigin("https://aegis.test"), func(c *gin.Context) { c.Status(http.StatusNoContent) })
			request := httptest.NewRequest(http.MethodPost, "/mutate", nil)
			request.Header.Set("Origin", test.origin)
			request.Header.Set("Referer", test.referer)
			request.Header.Set("Sec-Fetch-Site", test.fetchSite)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)
			if test.allowed {
				require.Equal(t, http.StatusNoContent, recorder.Code)
			} else {
				require.Equal(t, http.StatusForbidden, recorder.Code)
			}
		})
	}
}
