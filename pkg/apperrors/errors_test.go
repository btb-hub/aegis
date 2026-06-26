package apperrors

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestErrorCodes(t *testing.T) {
	t.Parallel()

	validation := Validation("bad input", map[string]any{"field": "locale"})
	require.Equal(t, "VALIDATION_ERROR", validation.Code)
	require.Equal(t, 400, validation.StatusCode)

	unauth := Unauthorized("missing session")
	require.Equal(t, "UNAUTHORIZED", unauth.Code)

	forbidden := Forbidden("viewer cannot mutate")
	require.Equal(t, "FORBIDDEN", forbidden.Code)

	locale := InvalidLocale()
	require.Equal(t, "INVALID_LOCALE", locale.Code)

	webhook := InvalidWebhookSecret()
	require.Equal(t, 401, webhook.StatusCode)

	notFound := NotFound("team")
	require.Equal(t, "NOT_FOUND", notFound.Code)
	require.Equal(t, 404, notFound.StatusCode)

	conflict := Conflict("duplicate")
	require.Equal(t, "CONFLICT", conflict.Code)
	require.Equal(t, 409, conflict.StatusCode)
}

func TestErrorString(t *testing.T) {
	t.Parallel()
	err := New("TEST", "something failed", 500)
	require.Equal(t, "something failed", err.Error())
}
