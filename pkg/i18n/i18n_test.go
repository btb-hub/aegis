package i18n

import (
	"os"
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

// I18N-EMPTY (i18n.go::T): catalog presence, not text length, controls fallback.
// Temporary catalogs exercise real loading and interpolation without changing
// shipped translations; an empty local value must override nonempty English.
func TestTranslationPresenceAndEmptyValues(t *testing.T) {
	t.Cleanup(func() { require.NoError(t, LoadMessages(messagesDir())) })
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "en.json"), []byte(`{"empty":"","empty_fallback":"","local_empty":"English","text":"Hello {{name}}"}`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ru.json"), []byte(`{"empty":"","local_empty":""}`), 0o600))
	require.NoError(t, LoadMessages(dir))
	for _, tc := range []struct {
		name, locale, key, want string
	}{
		{"empty English", "en", "empty", ""},
		{"empty Russian", "ru", "empty", ""},
		{"empty overrides English", "ru", "local_empty", ""},
		{"missing Russian falls back to empty English", "ru", "empty_fallback", ""},
		{"unknown locale preserves empty English", "de", "empty", ""},
		{"missing Russian interpolates English", "ru", "text", "Hello Alice"},
		{"missing everywhere returns key", "ru", "missing.key", "missing.key"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, T(tc.locale, tc.key, map[string]string{"name": "Alice"}))
		})
	}
}
