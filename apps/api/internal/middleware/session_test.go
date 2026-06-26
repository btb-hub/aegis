package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/aegis/aegis/pkg/db"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type sessionAuthMock struct {
	user db.User
	err  error
}

func (m *sessionAuthMock) CurrentUser(ctx context.Context, token string) (db.User, error) {
	if m.err != nil {
		return db.User{}, m.err
	}
	if token == "" {
		return db.User{}, apperrors.Unauthorized("missing session")
	}
	return m.user, nil
}

func TestRequireSessionSetsUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	user := db.User{ID: uuid.New(), Role: "member"}
	auth := &sessionAuthMock{user: user}
	r := gin.New()
	r.Use(RequireSession(auth))
	r.GET("/me", func(c *gin.Context) {
		got, ok := UserFromContext(c)
		require.True(t, ok)
		require.Equal(t, user.ID, got.ID)
		c.Status(http.StatusNoContent)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: "token"})
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)
}

func TestRequireSessionRejectsMissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth := &sessionAuthMock{user: db.User{Role: "member"}}
	r := gin.New()
	r.Use(RequireSession(auth))
	r.GET("/me", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRequireSessionBearerToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	user := db.User{ID: uuid.New(), Role: "admin"}
	auth := &sessionAuthMock{user: user}
	r := gin.New()
	r.Use(RequireSession(auth))
	r.GET("/me", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer abc")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)
}

func TestWriteErrorInternal(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	writeError(c, errors.New("boom"))
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUserFromContextMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	_, ok := UserFromContext(c)
	require.False(t, ok)
}

func TestSessionTokenEmpty(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	require.Empty(t, sessionToken(c))
}
