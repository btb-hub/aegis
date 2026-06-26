package i18n

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func messagesDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "messages")
}

func TestLoadAndTranslate(t *testing.T) {
	ResetForTests()
	dir := messagesDir()
	require.NoError(t, LoadMessages(dir))

	en := T("en", "page.acknowledge_button", nil)
	ru := T("ru", "page.acknowledge_button", nil)
	require.NotEqual(t, en, ru)
	require.Equal(t, "Acknowledge", en)
	require.Equal(t, "Подтвердить", ru)

	title := T("en", "page.incident_title", map[string]string{"id": "42"})
	require.Contains(t, title, "42")
}

func TestUnknownLocaleFallsBackToEnglish(t *testing.T) {
	ResetForTests()
	require.NoError(t, LoadMessages(messagesDir()))
	require.Equal(t, "Acknowledge", T("de", "page.acknowledge_button", nil))
}

func TestValidateParity(t *testing.T) {
	require.NoError(t, ValidateParity(messagesDir()))
}

func TestMissingKeyReturnsKey(t *testing.T) {
	ResetForTests()
	require.NoError(t, LoadMessages(messagesDir()))
	require.Equal(t, "missing.key", T("en", "missing.key", nil))
}
