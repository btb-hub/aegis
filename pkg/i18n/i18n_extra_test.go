package i18n

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadMessagesMissingFile(t *testing.T) {
	ResetForTests()
	err := LoadMessages(filepath.Join(t.TempDir(), "missing"))
	require.Error(t, err)
}

func TestValidateParityMismatch(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "en.json"), []byte(`{"a":"1"}`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ru.json"), []byte(`{}`), 0o600))
	require.Error(t, ValidateParity(dir))
}

func TestValidateParityMissingFile(t *testing.T) {
	require.Error(t, ValidateParity(t.TempDir()))
}
