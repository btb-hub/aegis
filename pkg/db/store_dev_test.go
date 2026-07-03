package db

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestUpsertDevUserIdempotent(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	store := NewStore(pool)

	user, err := store.UpsertDevUser(ctx, "dev@localhost", "Dev User", "admin", "en")
	require.NoError(t, err)
	require.Equal(t, "dev", user.Provider)
	require.Equal(t, "dev-local", user.ProviderSub)

	userAgain, err := store.UpsertDevUser(ctx, "dev@localhost", "Dev User", "admin", "en")
	require.NoError(t, err)
	require.Equal(t, user.ID, userAgain.ID)
}
