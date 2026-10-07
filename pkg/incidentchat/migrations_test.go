package incidentchat

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// SQL replay alone accepts duplicate version numbers, while golang-migrate
// rejects them before executing a migration. Check the same filename invariant.
func TestMigrationVersionsAreUniqueAndPaired(t *testing.T) {
	directory := filepath.Join("..", "..", "db", "migrations")
	for _, direction := range []string{"up", "down"} {
		paths, err := filepath.Glob(filepath.Join(directory, "*."+direction+".sql"))
		require.NoError(t, err)
		require.NotEmpty(t, paths)
		versions := make(map[string]string)
		for _, path := range paths {
			name := filepath.Base(path)
			version, _, ok := strings.Cut(name, "_")
			require.True(t, ok, "invalid migration filename: %s", name)
			require.Empty(t, versions[version], "duplicate migration version %s: %s and %s", version, versions[version], name)
			versions[version] = name
			other := "down"
			if direction == "down" {
				other = "up"
			}
			pair := strings.TrimSuffix(path, "."+direction+".sql") + "." + other + ".sql"
			require.FileExists(t, pair)
		}
	}
}
