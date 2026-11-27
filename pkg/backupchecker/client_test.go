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

const (
	token     = "0123456789abcdef"
	runningID = "77c0712f-c349-4e7c-8a8d-cbeabafa876f"
	failedID  = "c3a5d2e1-8b4f-4f0e-b6a2-91e7d4c0b222"
)

func fixture(t *testing.T, name string) string {
	t.Helper()
	raw, err := os.ReadFile("testdata/" + name)
	require.NoError(t, err)
	return string(raw)
}

func serverWithToken(t *testing.T, tok string, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		want := ""
		if tok != "" {
			want = "Bearer " + tok
		}
		require.Equal(t, want, r.Header.Get("Authorization"))
		handler(w, r)
	}))
	t.Cleanup(srv.Close)
	return New(config.BackupConfig{CheckerURL: srv.URL + "/", CheckerToken: tok}, nil)
}

func server(t *testing.T, handler http.HandlerFunc) *Client {
	return serverWithToken(t, token, handler)
}

func TestListChecks(t *testing.T) {
	body := fixture(t, "checks.json")
	client := serverWithToken(t, "", func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "GET /v1/checks", r.Method+" "+r.URL.Path)
		_, _ = io.WriteString(w, body)
	})
	checks, err := client.ListChecks(context.Background())
	require.NoError(t, err)
	require.Len(t, checks, 3)

	running := checks[0]
	require.Equal(t, runningID, running.ID)
	require.Equal(t, "gamepass-prod-pg", running.Backup)
	require.Equal(t, "postgres", running.Engine)
	require.Equal(t, StatusRunning, running.Status)
	require.Equal(t, 82200000, running.StartedAt.Nanosecond())
	require.False(t, running.Finished())
	require.Nil(t, running.ReplicasMatch)

	failed := checks[1]
	require.Equal(t, StatusFailed, failed.Status, "sorted newest first")
	require.False(t, *failed.ReplicasMatch)
	require.Equal(t, "sha256 differs between sources", failed.Message)
	require.Nil(t, failed.RestoreOK, "list items have no restore result")

	require.Equal(t, StatusPassed, checks[2].Status)
}

func TestGetCheckDetails(t *testing.T) {
	client := server(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "GET /v1/checks/" + runningID:
			_, _ = io.WriteString(w, fixture(t, "check_success.json"))
		case "GET /v1/checks/" + failedID:
			_, _ = io.WriteString(w, fixture(t, "check_failed.json"))
		default:
			http.NotFound(w, r)
		}
	})
	ctx := context.Background()
	check, err := client.GetCheck(ctx, runningID)
	require.NoError(t, err)
	require.Equal(t, StatusPassed, check.Status)
	require.True(t, *check.ReplicasMatch)
	require.True(t, *check.RestoreOK)
	require.True(t, *check.ManifestCountsMatch)
	require.Empty(t, check.Message)
	require.Equal(t, time.Date(2026, 11, 26, 17, 20, 11, 0, time.UTC), *check.FinishedAt)

	check, err = client.GetCheck(ctx, failedID)
	require.NoError(t, err)
	require.Equal(t, StatusFailed, check.Status)
	require.False(t, *check.ReplicasMatch)
	require.True(t, *check.RestoreOK)
	require.False(t, *check.ManifestCountsMatch)
	require.Equal(t, "sha256 differs between sources; minio-dc: object not found", check.Message)

	_, err = client.GetCheck(ctx, "missing")
	require.ErrorIs(t, err, ErrNotFound)
	_, err = client.GetCheck(ctx, "../etc")
	require.ErrorIs(t, err, ErrNotFound)
}

func TestGetCheckRejectsOtherID(t *testing.T) {
	client := server(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, fixture(t, "check_success.json"))
	})
	_, err := client.GetCheck(context.Background(), failedID)
	require.ErrorIs(t, err, ErrInvalidResponse)
}

func TestRunChecks(t *testing.T) {
	calls := 0
	client := server(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		require.Equal(t, "POST /v1/checks/run", r.Method+" "+r.URL.Path)
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))
		raw, _ := io.ReadAll(r.Body)
		require.JSONEq(t, `{}`, string(raw))
		if calls == 2 {
			w.WriteHeader(http.StatusConflict)
			return
		}
		w.WriteHeader(http.StatusAccepted)
		_, _ = io.WriteString(w, `{"anything": "goes"}`)
	})
	require.NoError(t, client.RunChecks(context.Background()))
	require.ErrorIs(t, client.RunChecks(context.Background()), ErrRunning)
}

func TestErrors(t *testing.T) {
	ctx := context.Background()
	_, err := New(config.BackupConfig{}, nil).ListChecks(ctx)
	require.ErrorIs(t, err, ErrNotConfigured)
	_, err = New(config.BackupConfig{CheckerURL: "http://%zz"}, nil).ListChecks(ctx)
	require.ErrorIs(t, err, ErrNotConfigured)
	_, err = New(config.BackupConfig{CheckerURL: "http://127.0.0.1:1"}, nil).ListChecks(ctx)
	require.ErrorIs(t, err, ErrUnavailable)

	item := func(fields string) string {
		return `[{"id":"a","target_id":"x","engine":"postgres","status":"success","started_at":"2026-11-25T00:00:00Z","finished_at":"2026-11-25T01:00:00Z"` + fields + `}]`
	}
	cases := map[string]struct {
		status int
		body   string
		want   error
	}{
		"server error":    {http.StatusInternalServerError, `{}`, ErrUnavailable},
		"redirect":        {http.StatusFound, ``, ErrUnavailable},
		"conflict":        {http.StatusConflict, `{}`, ErrRunning},
		"not found":       {http.StatusNotFound, `{}`, ErrNotFound},
		"not json":        {http.StatusOK, `<html>`, ErrInvalidResponse},
		"trailing":        {http.StatusOK, `[] {}`, ErrInvalidResponse},
		"object":          {http.StatusOK, `{"items":[]}`, ErrInvalidResponse},
		"null":            {http.StatusOK, `null`, ErrInvalidResponse},
		"bad target":      {http.StatusOK, item(`,"target_id":"a b"`), ErrInvalidResponse},
		"no finished_at":  {http.StatusOK, item(`,"finished_at":null`), ErrInvalidResponse},
		"too large":       {http.StatusOK, `[` + strings.Repeat(`{},`, maxResponseBytes/3) + `{}]`, ErrInvalidResponse},
		"too many checks": {http.StatusOK, `[` + strings.Repeat(`{},`, maxChecks) + `{}]`, ErrInvalidResponse},
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
	ok := server(t, func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, item("")) })
	checks, err := ok.ListChecks(ctx)
	require.NoError(t, err)
	require.Equal(t, StatusPassed, checks[0].Status)
}

func TestNormalize(t *testing.T) {
	var base wireCheck
	require.NoError(t, json.Unmarshal([]byte(fixture(t, "check_success.json")), &base))
	_, err := base.normalize()
	require.NoError(t, err)

	invalid := []func(*wireCheck){
		func(c *wireCheck) { c.ID = "" },
		func(c *wireCheck) { c.ID = "a b" },
		func(c *wireCheck) { c.ID = strings.Repeat("a", 129) },
		func(c *wireCheck) { c.TargetID = "" },
		func(c *wireCheck) { c.TargetID = "-gamepass" },
		func(c *wireCheck) { c.Engine = strings.Repeat("e", maxEngineLength+1) },
		func(c *wireCheck) { c.Status = " " },
		func(c *wireCheck) { c.StartedAt = time.Time{} },
		func(c *wireCheck) { c.FinishedAt = nil },
	}
	for i, m := range invalid {
		c := base
		m(&c)
		_, err := c.normalize()
		require.Error(t, err, "case %d", i)
	}

	long := base
	long.Error = json.RawMessage(`"` + strings.Repeat("я", maxMessageRunes+10) + `"`)
	check, err := long.normalize()
	require.NoError(t, err)
	require.Equal(t, maxMessageRunes, len([]rune(check.Message)))

	structured := base
	structured.Restore = &wireRestore{OK: boolPtr(false), Error: json.RawMessage(`{"code": "pg_restore", "exit": 1}`)}
	check, err = structured.normalize()
	require.NoError(t, err)
	require.False(t, *check.RestoreOK)
	require.Nil(t, check.ManifestCountsMatch, "no inventory, counts not compared")
	require.Equal(t, `{"code":"pg_restore","exit":1}`, check.Message)

	require.True(t, ValidBackupName("gamepass-prod-pg"))
	require.False(t, ValidCheckID("a%2f"))
}

func TestNormalizeStatus(t *testing.T) {
	for raw, want := range map[string]string{
		"success": StatusPassed, "SUCCESS": StatusPassed, "passed": StatusPassed,
		"running": StatusRunning, "queued": StatusRunning,
		"failed": StatusFailed, "error": StatusFailed, "timeout": StatusFailed,
	} {
		require.Equal(t, want, normalizeStatus(raw, true), raw)
	}
	require.Equal(t, StatusFailed, normalizeStatus("partial", true), "unknown finished status never passes")
	require.Equal(t, StatusRunning, normalizeStatus("restoring", false))
}

func TestManifestMatch(t *testing.T) {
	present := func(m wireManifest) wireDump { m.Present = true; return wireDump{Manifest: &m} }
	require.Nil(t, manifestMatch(nil))
	require.Nil(t, manifestMatch(&wireRestore{}))
	require.True(t, *manifestMatch(&wireRestore{Inventory: map[string]wireDump{"a": present(wireManifest{})}}))
	require.Nil(t, manifestMatch(&wireRestore{Inventory: map[string]wireDump{"a": present(wireManifest{}), "b": {}}}))
	for name, m := range map[string]wireManifest{
		"count":   {MismatchCount: 2},
		"list":    {Mismatches: []json.RawMessage{json.RawMessage(`{}`)}},
		"extra":   {ExtraTables: []string{"public.tmp"}},
		"missing": {MissingTables: []string{"public.users"}},
	} {
		got := manifestMatch(&wireRestore{Inventory: map[string]wireDump{"a": present(m), "b": {}}})
		require.False(t, *got, name)
	}
	require.Equal(t, "", errorText(nil))
	require.Equal(t, "", errorText(json.RawMessage(`null`)))
	require.Equal(t, "", errorText(json.RawMessage(`{bad`)))
}
