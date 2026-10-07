package db

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aegis/aegis/pkg/sessiontoken"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestPagingConnectionsPostgres(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	store := NewStore(pool)
	user := func(t *testing.T) User {
		id := uuid.NewString()
		result, err := store.ResolveOIDCLogin(ctx, OIDCLoginInput{Provider: "google", ProviderSub: id, Email: id + "@paging.test", DisplayName: "Paging test"})
		require.NoError(t, err)
		t.Cleanup(func() {
			_, _ = pool.Exec(ctx, `DELETE FROM audit_log WHERE resource_id = $1`, result.User.ID)
			_, _ = pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, result.User.ID)
		})
		return result.User
	}
	attempt := func(t *testing.T, u User, provider string) PagingAuthorization {
		token, hash, err := sessiontoken.New()
		require.NoError(t, err)
		_, err = store.CreateSession(ctx, u.ID, hash, time.Now().Add(time.Hour))
		require.NoError(t, err)
		a := PagingAuthorization{StateHash: sessiontoken.Hash(uuid.NewString()), UserID: u.ID,
			SessionHash: sessiontoken.Hash(token), Provider: provider, Nonce: uuid.NewString(), ExpiresAt: time.Now().Add(5 * time.Minute)}
		require.NoError(t, store.CreatePagingAuthorization(ctx, a))
		return a
	}
	claim := func(t *testing.T, a PagingAuthorization) PagingAuthorization {
		claimed, err := store.ClaimPagingAuthorization(ctx, a.StateHash, a.SessionHash, a.Provider)
		require.NoError(t, err)
		require.Equal(t, a.UserID, claimed.UserID)
		require.Equal(t, a.Nonce, claimed.Nonce)
		return claimed
	}
	t.Run("replace and disconnect preserve sign in and profile", func(t *testing.T) {
		u := user(t)
		originalIdentities, err := store.ListUserIdentities(ctx, u.ID)
		require.NoError(t, err)
		a := claim(t, attempt(t, u, "slack"))
		require.NoError(t, store.CompletePagingAuthorization(ctx, a, "U"+uuid.NewString()))
		before, err := store.GetUserByID(ctx, u.ID)
		require.NoError(t, err)
		a = claim(t, attempt(t, u, "slack"))
		stillConnected, err := store.GetUserByID(ctx, u.ID)
		require.NoError(t, err)
		require.Equal(t, before.SlackUserID, stillConnected.SlackUserID)
		require.NoError(t, store.CompletePagingAuthorization(ctx, a, "U"+uuid.NewString()))
		after, err := store.GetUserByID(ctx, u.ID)
		require.NoError(t, err)
		require.NotEqual(t, before.SlackUserID, after.SlackUserID)
		require.NoError(t, store.DisconnectPagingIdentity(ctx, u.ID, "slack"))
		after, err = store.GetUserByID(ctx, u.ID)
		require.NoError(t, err)
		require.Nil(t, after.SlackUserID)
		require.Equal(t, u.Email, after.Email)
		require.Equal(t, u.Role, after.Role)
		require.Equal(t, u.DisplayName, after.DisplayName)
		identities, err := store.ListUserIdentities(ctx, u.ID)
		require.NoError(t, err)
		require.Equal(t, originalIdentities, identities)
		login, err := store.ResolveOIDCLogin(ctx, OIDCLoginInput{Provider: u.Provider, ProviderSub: u.ProviderSub, Email: u.Email, SlackUserID: "U" + uuid.NewString()})
		require.NoError(t, err)
		require.Nil(t, login.User.SlackUserID, "later sign-in must not reconnect")
		var events int
		require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM audit_log WHERE resource_id = $1 AND action LIKE 'paging.%'`, u.ID).Scan(&events))
		require.Equal(t, 3, events)
	})
	t.Run("reject replay session changes and expiry", func(t *testing.T) {
		u := user(t)
		a := attempt(t, u, "express")
		_, err := store.ClaimPagingAuthorization(ctx, a.StateHash, "other-session", a.Provider)
		require.Error(t, err)
		_, err = store.ClaimPagingAuthorization(ctx, a.StateHash, a.SessionHash, "slack")
		require.Error(t, err)
		a = claim(t, a)
		_, err = store.ClaimPagingAuthorization(ctx, a.StateHash, a.SessionHash, a.Provider)
		require.Error(t, err)
		require.NoError(t, store.CompletePagingAuthorization(ctx, a, uuid.NewString()))
		require.Error(t, store.CompletePagingAuthorization(ctx, a, uuid.NewString()))
		a = attempt(t, u, "express")
		_, err = pool.Exec(ctx, `UPDATE paging_authorizations SET expires_at = now() - interval '1 second' WHERE state_hash = $1`, a.StateHash)
		require.NoError(t, err)
		_, err = store.ClaimPagingAuthorization(ctx, a.StateHash, a.SessionHash, a.Provider)
		require.Error(t, err)
		a = claim(t, attempt(t, u, "express"))
		require.NoError(t, store.DeleteSession(ctx, a.SessionHash))
		require.Error(t, store.CompletePagingAuthorization(ctx, a, uuid.NewString()))
		require.Error(t, store.CreatePagingAuthorization(ctx, a))
	})
	t.Run("disconnect and restart cancel in flight exchanges and link codes", func(t *testing.T) {
		u := user(t)
		a := claim(t, attempt(t, u, "express"))
		code, err := store.CreateExpressLinkCode(ctx, u.ID, time.Minute)
		require.NoError(t, err)
		require.NoError(t, store.DisconnectPagingIdentity(ctx, u.ID, "express"))
		require.Error(t, store.CompletePagingAuthorization(ctx, a, uuid.NewString()))
		_, err = store.RedeemExpressLinkCode(ctx, code, uuid.New())
		require.Error(t, err)
		a = claim(t, attempt(t, u, "express"))
		replacement := attempt(t, u, "express")
		require.Error(t, store.CompletePagingAuthorization(ctx, a, uuid.NewString()))
		replacement = claim(t, replacement)
		require.NoError(t, store.CompletePagingAuthorization(ctx, replacement, uuid.NewString()))
		code, err = store.CreateExpressLinkCode(ctx, u.ID, time.Minute)
		require.NoError(t, err)
		a = claim(t, attempt(t, u, "express"))
		_, err = store.RedeemExpressLinkCode(ctx, code, uuid.New())
		require.NoError(t, err)
		require.Error(t, store.CompletePagingAuthorization(ctx, a, uuid.NewString()))
	})
	t.Run("concurrent assignments and legacy writes cannot steal identity", func(t *testing.T) {
		first, second := user(t), user(t)
		a, b := claim(t, attempt(t, first, "express")), claim(t, attempt(t, second, "express"))
		huid := uuid.New()
		var wg sync.WaitGroup
		outcomes := make(chan error, 2)
		for _, current := range []PagingAuthorization{a, b} {
			wg.Add(1)
			go func(current PagingAuthorization) {
				defer wg.Done()
				outcomes <- store.CompletePagingAuthorization(ctx, current, huid.String())
			}(current)
		}
		wg.Wait()
		close(outcomes)
		success := 0
		for err := range outcomes {
			if err == nil {
				success++
			}
		}
		require.Equal(t, 1, success)
		owner, err := store.GetUserByExpressHuid(ctx, huid)
		require.NoError(t, err)
		loser := first
		if owner.ID == first.ID {
			loser = second
		}
		_, err = store.UpdateUserExpressHuid(ctx, loser.ID, huid)
		require.ErrorContains(t, err, "already connected")
		code, err := store.CreateExpressLinkCode(ctx, loser.ID, time.Minute)
		require.NoError(t, err)
		_, err = store.RedeemExpressLinkCode(ctx, code, huid)
		require.ErrorContains(t, err, "already connected")
		unchanged, err := store.GetUserByID(ctx, loser.ID)
		require.NoError(t, err)
		require.False(t, unchanged.ExpressUserHuid.Valid)
	})
	t.Run("login backfill still works for unmanaged users and checks ownership", func(t *testing.T) {
		u, other := user(t), user(t)
		slack := "U" + uuid.NewString()
		result, err := store.ResolveOIDCLogin(ctx, OIDCLoginInput{Provider: u.Provider, ProviderSub: u.ProviderSub, SlackUserID: slack})
		require.NoError(t, err)
		require.Equal(t, slack, *result.User.SlackUserID)
		_, err = store.ResolveOIDCLogin(ctx, OIDCLoginInput{Provider: other.Provider, ProviderSub: other.ProviderSub, SlackUserID: slack})
		require.ErrorContains(t, err, "already connected")
		pending := claim(t, attempt(t, other, "slack"))
		require.ErrorContains(t, store.CompletePagingAuthorization(ctx, pending, slack), "already connected")
	})
}

func TestPagingMigrationPreservesExistingIDs(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	require.NoError(t, err)
	defer pool.Close()
	connection, err := pool.Acquire(ctx)
	require.NoError(t, err)
	defer connection.Release()
	schema := "paging_migration_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	_, err = connection.Exec(ctx, `CREATE SCHEMA `+schema)
	require.NoError(t, err)
	defer connection.Exec(ctx, `DROP SCHEMA `+schema+` CASCADE`)
	_, err = connection.Exec(ctx, `SET search_path TO `+schema)
	require.NoError(t, err)
	_, err = connection.Exec(ctx, `CREATE TABLE users (id uuid PRIMARY KEY, slack_user_id text, express_user_huid uuid);
        CREATE TABLE sessions (token_hash text UNIQUE NOT NULL);`)
	require.NoError(t, err)
	userID, huid := uuid.New(), uuid.New()
	_, err = connection.Exec(ctx, `INSERT INTO users VALUES ($1, 'ULEGACY', $2), ($3, 'ULEGACY', $2)`, userID, huid, uuid.New())
	require.NoError(t, err)
	up, err := os.ReadFile(filepath.Join("..", "..", "db", "migrations", "000020_paging_connections.up.sql"))
	require.NoError(t, err)
	down, err := os.ReadFile(filepath.Join("..", "..", "db", "migrations", "000020_paging_connections.down.sql"))
	require.NoError(t, err)
	for _, migration := range [][]byte{up, down, up} {
		_, err = connection.Exec(ctx, string(migration))
		require.NoError(t, err)
		var slack string
		var actualHUID uuid.UUID
		require.NoError(t, connection.QueryRow(ctx, `SELECT slack_user_id, express_user_huid FROM users WHERE id = $1`, userID).Scan(&slack, &actualHUID))
		require.Equal(t, "ULEGACY", slack)
		require.Equal(t, huid, actualHUID)
		var count int
		require.NoError(t, connection.QueryRow(ctx, `SELECT count(*) FROM users`).Scan(&count))
		require.Equal(t, 2, count, "legacy duplicates must not be deleted")
	}
}
