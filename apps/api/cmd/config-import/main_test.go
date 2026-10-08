package main

import (
	"bytes"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestImporterArgumentsAndErrorsNeverEchoValues(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	path := filepath.Join(t.TempDir(), "fixture.env")
	for _, tc := range []struct {
		args []string
		data string
	}{
		{[]string{}, ""},
		{[]string{"--dry-run=private-secret-value"}, ""},
		{[]string{"--env-file", path, "--dry-run", "--apply"}, ""},
		{[]string{"--env-file", path, "--apply", "private-secret-value"}, ""},
		{[]string{"--env-file", path + "missing", "--dry-run"}, ""},
		{[]string{"--env-file", path, "--dry-run"}, "BAD KEY=private-secret-value"},
		{[]string{"--env-file", path, "--dry-run"}, "SESSION_SECRET=private-secret-value"},
		{[]string{"--env-file", path, "--dry-run"}, "DATABASE_URL='postgres://%zz/private-secret-value'"},
	} {
		require.NoError(t, os.WriteFile(path, []byte(tc.data), 0600))
		var out bytes.Buffer
		err := run(tc.args, &out)
		require.Error(t, err)
		require.NotContains(t, out.String()+err.Error(), "private-secret-value")
	}
}

func TestImporterConnectionFailureIsRedactedAndDoesNotExecuteEnv(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	path := filepath.Join(t.TempDir(), "fixture.env")
	require.NoError(t, os.WriteFile(path, []byte("DATABASE_URL='postgres://private-user:private-password@127.0.0.1:1/aegis?connect_timeout=1'\nWEBHOOK_SECRET=$(private-command)\n"), 0600))
	var out bytes.Buffer
	err := run([]string{"--env-file", path, "--dry-run"}, &out)
	require.Error(t, err)
	require.False(t, strings.Contains(out.String()+err.Error(), "private-"))
}
