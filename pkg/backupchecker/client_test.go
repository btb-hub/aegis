package backupchecker

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aegis/aegis/pkg/config"
	"github.com/stretchr/testify/require"
)

const token = "0123456789abcdef"

func fixture(t *testing.T, name string) string {
	t.Helper()
	raw, err := os.ReadFile("testdata/" + name)
	require.NoError(t, err)
	return string(raw)
}

func server(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer "+token, r.Header.Get("Authorization"))
		handler(w, r)
	}))
	t.Cleanup(srv.Close)
	return New(config.BackupConfig{CheckerURL: srv.URL + "/", CheckerToken: token}, nil)
}

func TestListChecks(t *testing.T) {
	body := fixture(t, "checks.json")
	client := server(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "GET /v1/checks", r.Method+" "+r.URL.Path)
		_, _ = io.WriteString(w, body)
	})
	checks, err := client.ListChecks(context.Background())
	require.NoError(t, err)
	require.Len(t, checks, 4)
	require.True(t, checks[0].Finished())
	require.False(t, *checks[1].ManifestCountsMatch)
	require.False(t, checks[3].Finished())
	require.Nil(t, checks[3].RestoreOK)
}

func TestGetAndRunCheck(t *testing.T) {
	running := fixture(t, "check_running.json")
	client := server(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "GET /v1/checks/chk_manual_1":
			_, _ = io.WriteString(w, running)
		case "POST /v1/checks/run":
			require.Equal(t, "application/json", r.Header.Get("Content-Type"))
			var in map[string]string
			require.NoError(t, json.NewDecoder(r.Body).Decode(&in))
			require.Contains(t, []string{"gamepass", "nish"}, in["backup"])
			w.WriteHeader(http.StatusAccepted)
			_, _ = io.WriteString(w, running)
		default:
			http.NotFound(w, r)
		}
	})
	ctx := context.Background()
	check, err := client.GetCheck(ctx, "chk_manual_1")
	require.NoError(t, err)
	require.Equal(t, StatusRunning, check.Status)
	_, err = client.GetCheck(ctx, "missing")
	require.ErrorIs(t, err, ErrNotFound)
	_, err = client.GetCheck(ctx, "../etc")
	require.ErrorIs(t, err, ErrNotFound)

	check, err = client.RunCheck(ctx, "gamepass")
	require.NoError(t, err)
	require.Equal(t, "manual", check.Trigger)
	_, err = client.RunCheck(ctx, "Bad Name")
	require.ErrorIs(t, err, ErrNotFound)
	_, err = client.RunCheck(ctx, "nish")
	require.ErrorIs(t, err, ErrInvalidResponse, "checker answered for another backup")
}

func TestErrors(t *testing.T) {
	ctx := context.Background()
	_, err := New(config.BackupConfig{}, nil).ListChecks(ctx)
	require.ErrorIs(t, err, ErrNotConfigured)
	_, err = New(config.BackupConfig{CheckerURL: "http://%zz", CheckerToken: token}, nil).ListChecks(ctx)
	require.ErrorIs(t, err, ErrNotConfigured)
	_, err = New(config.BackupConfig{CheckerURL: "http://127.0.0.1:1", CheckerToken: token}, nil).ListChecks(ctx)
	require.ErrorIs(t, err, ErrUnavailable)

	cases := map[string]struct {
		status int
		body   string
		want   error
	}{
		"server error":  {http.StatusInternalServerError, `{}`, ErrUnavailable},
		"redirect":      {http.StatusFound, ``, ErrUnavailable},
		"conflict":      {http.StatusConflict, `{}`, ErrRunning},
		"not json":      {http.StatusOK, `<html>`, ErrInvalidResponse},
		"trailing":      {http.StatusOK, `{"items":[]} {}`, ErrInvalidResponse},
		"missing items": {http.StatusOK, `{}`, ErrInvalidResponse},
		"bad status":    {http.StatusOK, `{"items":[{"id":"a","backup":"x","status":"ok","started_at":"2026-11-25T00:00:00Z"}]}`, ErrInvalidResponse},
		"too large":     {http.StatusOK, `{"items":[` + strings.Repeat(`{},`, maxResponseBytes/3) + `{}]}`, ErrInvalidResponse},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			client := server(t, func(w http.ResponseWriter, r *http.Request) {
				if tc.status == http.StatusFound {
					w.Header().Set("Location", "https://elsewhere.example/")
				}
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, tc.body)
			})
			_, err := client.ListChecks(ctx)
			require.True(t, errors.Is(err, tc.want), "got %v", err)
		})
	}
}

func TestCheckValidation(t *testing.T) {
	var base Check
	require.NoError(t, json.Unmarshal([]byte(fixture(t, "check_running.json")), &base))
	require.NoError(t, base.validate())
	mutate := []func(*Check){
		func(c *Check) { c.ID = "" },
		func(c *Check) { c.ID = "a b" },
		func(c *Check) { c.ID = strings.Repeat("a", 129) },
		func(c *Check) { c.Backup = "Gamepass" },
		func(c *Check) { c.Status = "ok" },
		func(c *Check) { c.Trigger = "cron" },
		func(c *Check) { c.StartedAt = time.Time{} },
		func(c *Check) { c.Status = StatusPassed },
		func(c *Check) { c.Message = strings.Repeat("x", 2001) },
	}
	for i, m := range mutate {
		c := base
		m(&c)
		require.Error(t, c.validate(), "case %d", i)
	}
	require.True(t, ValidBackupName("nish"))
	require.False(t, ValidCheckID("a%2f"))
}
