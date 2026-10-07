package middleware

import (
	"net/url"
	"strings"

	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/gin-gonic/gin"
)

// RequireSameOrigin fails closed for browser mutations, including missing headers.
func RequireSameOrigin(publicURL string) gin.HandlerFunc {
	expected, expectedErr := url.Parse(publicURL)
	return func(c *gin.Context) {
		raw := c.GetHeader("Origin")
		if raw == "" {
			raw = c.GetHeader("Referer")
		}
		origin, err := url.Parse(raw)
		if expectedErr != nil || expected.Host == "" || err != nil || origin.Host == "" || origin.User != nil ||
			!strings.EqualFold(origin.Scheme, expected.Scheme) || !strings.EqualFold(origin.Host, expected.Host) || c.GetHeader("Sec-Fetch-Site") == "cross-site" {
			writeError(c, apperrors.Forbidden("request must originate from Aegis"))
			c.Abort()
			return
		}
		c.Next()
	}
}
