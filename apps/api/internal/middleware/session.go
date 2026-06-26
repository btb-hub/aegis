package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/aegis/aegis/pkg/db"
	"github.com/gin-gonic/gin"
)

const (
	sessionCookie = "aegis_session"
	userKey       = "user"
)

type SessionUserResolver interface {
	CurrentUser(ctx context.Context, token string) (db.User, error)
}

func RequireSession(auth SessionUserResolver) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := sessionToken(c)
		user, err := auth.CurrentUser(c.Request.Context(), token)
		if err != nil {
			writeError(c, err)
			c.Abort()
			return
		}
		c.Set(userKey, user)
		c.Set(userRoleKey, user.Role)
		c.Next()
	}
}

func UserFromContext(c *gin.Context) (db.User, bool) {
	value, ok := c.Get(userKey)
	if !ok {
		return db.User{}, false
	}
	user, ok := value.(db.User)
	return user, ok
}

func sessionToken(c *gin.Context) string {
	token, _ := c.Cookie(sessionCookie)
	if token != "" {
		return token
	}
	auth := c.GetHeader("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}

func writeError(c *gin.Context, err error) {
	var appErr *apperrors.Error
	if errors.As(err, &appErr) {
		c.JSON(appErr.StatusCode, gin.H{
			"code":    appErr.Code,
			"message": appErr.Message,
			"details": appErr.Details,
		})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{
		"code":    "INTERNAL_ERROR",
		"message": "unexpected error",
	})
}
