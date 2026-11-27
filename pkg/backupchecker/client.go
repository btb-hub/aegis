// Package backupchecker is the Aegis client for the backup-checker service.
// The contract is documented in docs/integrations/backup-checker.md.
package backupchecker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/aegis/aegis/pkg/config"
)

// Normalized statuses Aegis uses for a check run.
const (
	StatusRunning = "running"
	StatusPassed  = "passed"
	StatusFailed  = "failed"

	maxResponseBytes = 4 << 20
	maxChecks        = 10000
	maxMessageRunes  = 2000
	maxEngineLength  = 64
)

var (
	ErrNotConfigured   = errors.New("backup checker is not configured")
	ErrUnavailable     = errors.New("backup checker is unavailable")
	ErrInvalidResponse = errors.New("backup checker returned an invalid response")
	ErrNotFound        = errors.New("backup check not found")
	ErrRunning         = errors.New("a backup check is already running")

	targetName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)
)

// Check is one backup-checker run, normalized for Aegis. Backup is the checker's target_id.
// RestoreOK and ManifestCountsMatch are only known from the run details (GetCheck); ListChecks
// leaves them nil.
type Check struct {
	ID                  string     `json:"id"`
	Backup              string     `json:"backup"`
	Engine              string     `json:"engine"`
	Status              string     `json:"status"`
	StartedAt           time.Time  `json:"started_at"`
	FinishedAt          *time.Time `json:"finished_at"`
	ReplicasMatch       *bool      `json:"replicas_match"`
	RestoreOK           *bool      `json:"restore_ok"`
	ManifestCountsMatch *bool      `json:"manifest_counts_match"`
	Message             string     `json:"message,omitempty"`
}

// Finished reports whether the run has a final result.
func (c Check) Finished() bool { return c.Status == StatusPassed || c.Status == StatusFailed }

// ValidBackupName reports whether name is a usable checker target_id.
func ValidBackupName(name string) bool { return targetName.MatchString(name) }

// ValidCheckID reports whether id is safe to use as a path segment and database key.
func ValidCheckID(id string) bool {
	if id == "" || len(id) > 128 {
		return false
	}
	for _, r := range id {
		if unicode.IsControl(r) || unicode.IsSpace(r) || r == '/' || r == '?' || r == '#' || r == '%' {
			return false
		}
	}
	return true
}

// wireCheck is the checker's JSON for a run. List items carry the top-level fields only;
// the details endpoint adds sources and restore.
type wireCheck struct {
	ID           string          `json:"id"`
	TargetID     string          `json:"target_id"`
	Engine       string          `json:"engine"`
	Status       string          `json:"status"`
	StartedAt    time.Time       `json:"started_at"`
	FinishedAt   *time.Time      `json:"finished_at"`
	SourcesMatch *bool           `json:"sources_match"`
	Error        json.RawMessage `json:"error"`
	Sources      []wireSource    `json:"sources"`
	Restore      *wireRestore    `json:"restore"`
}

type wireSource struct {
	Name  string          `json:"name"`
	OK    *bool           `json:"ok"`
	Error json.RawMessage `json:"error"`
}

type wireRestore struct {
	OK        *bool               `json:"ok"`
	Error     json.RawMessage     `json:"error"`
	Inventory map[string]wireDump `json:"inventory"`
}

type wireDump struct {
	Manifest *wireManifest `json:"manifest"`
}

type wireManifest struct {
	Present       bool              `json:"present"`
	MismatchCount int               `json:"mismatch_count"`
	Mismatches    []json.RawMessage `json:"mismatches"`
	ExtraTables   []string          `json:"extra_tables"`
	MissingTables []string          `json:"missing_tables"`
}

// normalizeStatus maps checker statuses to running, passed or failed. Unknown statuses count
// as failed once the run has finished, so a new failure kind never shows as passed.
func normalizeStatus(raw string, finished bool) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "success", "succeeded", "passed", "ok":
		return StatusPassed
	case "running", "pending", "queued", "started", "in_progress":
		return StatusRunning
	case "failed", "failure", "error", "errored", "timeout", "cancelled", "canceled":
		return StatusFailed
	}
	if finished {
		return StatusFailed
	}
	return StatusRunning
}

// errorText turns the checker's error value (null, a string or any JSON) into display text.
func errorText(raw json.RawMessage) string {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return strings.TrimSpace(s)
	}
	var compact bytes.Buffer
	if json.Compact(&compact, raw) == nil {
		return compact.String()
	}
	return ""
}

func truncateRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n-1]) + "…"
}

// manifestMatch is false when any dump differs from its manifest, nil when a dump has no
// manifest (counts were not compared) and true when every dump matches.
func manifestMatch(r *wireRestore) *bool {
	if r == nil || len(r.Inventory) == 0 {
		return nil
	}
	unknown := false
	for _, dump := range r.Inventory {
		m := dump.Manifest
		if m == nil || !m.Present {
			unknown = true
			continue
		}
		if m.MismatchCount != 0 || len(m.Mismatches) != 0 || len(m.ExtraTables) != 0 || len(m.MissingTables) != 0 {
			return boolPtr(false)
		}
	}
	if unknown {
		return nil
	}
	return boolPtr(true)
}

func boolPtr(v bool) *bool { return &v }

func (w wireCheck) normalize() (Check, error) {
	switch {
	case !ValidCheckID(w.ID):
		return Check{}, fmt.Errorf("invalid check id")
	case !ValidBackupName(w.TargetID):
		return Check{}, fmt.Errorf("invalid target_id")
	case len(w.Engine) > maxEngineLength:
		return Check{}, fmt.Errorf("engine too long")
	case strings.TrimSpace(w.Status) == "":
		return Check{}, fmt.Errorf("missing status")
	case w.StartedAt.IsZero():
		return Check{}, fmt.Errorf("missing started_at")
	}
	c := Check{ID: w.ID, Backup: w.TargetID, Engine: w.Engine, StartedAt: w.StartedAt, FinishedAt: w.FinishedAt,
		Status: normalizeStatus(w.Status, w.FinishedAt != nil), ReplicasMatch: w.SourcesMatch, ManifestCountsMatch: manifestMatch(w.Restore)}
	if c.Finished() && c.FinishedAt == nil {
		return Check{}, fmt.Errorf("missing finished_at")
	}
	messages := []string{errorText(w.Error)}
	if w.Restore != nil {
		c.RestoreOK = w.Restore.OK
		messages = append(messages, errorText(w.Restore.Error))
	}
	for _, source := range w.Sources {
		if text := errorText(source.Error); text != "" {
			messages = append(messages, source.Name+": "+text)
		}
	}
	parts := messages[:0]
	for _, m := range messages {
		if m != "" {
			parts = append(parts, m)
		}
	}
	c.Message = truncateRunes(strings.Join(parts, "; "), maxMessageRunes)
	return c, nil
}

// Client calls the backup-checker HTTP API. The bearer token is optional: in-cluster
// checkers usually rely on network policy instead.
type Client struct {
	base  string
	token string
	http  *http.Client
}

// New returns a client for cfg. A nil httpClient uses a 10 second timeout and
// never follows redirects, so a token only reaches the configured host.
func New(cfg config.BackupConfig, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	return &Client{base: strings.TrimRight(cfg.CheckerURL, "/"), token: cfg.CheckerToken, http: httpClient}
}

// ListChecks returns runs from GET /v1/checks, newest first. Restore and manifest results are
// not part of the list; read them with GetCheck.
func (c *Client) ListChecks(ctx context.Context) ([]Check, error) {
	var body []wireCheck
	if err := c.do(ctx, http.MethodGet, "/v1/checks", nil, &body); err != nil {
		return nil, err
	}
	if body == nil || len(body) > maxChecks {
		return nil, ErrInvalidResponse
	}
	checks := make([]Check, 0, len(body))
	for _, raw := range body {
		check, err := raw.normalize()
		if err != nil {
			return nil, ErrInvalidResponse
		}
		checks = append(checks, check)
	}
	sort.SliceStable(checks, func(i, j int) bool { return checks[i].StartedAt.After(checks[j].StartedAt) })
	return checks, nil
}

// GetCheck returns one run from GET /v1/checks/{id}, including restore and manifest results.
func (c *Client) GetCheck(ctx context.Context, id string) (Check, error) {
	if !ValidCheckID(id) {
		return Check{}, ErrNotFound
	}
	var raw wireCheck
	if err := c.do(ctx, http.MethodGet, "/v1/checks/"+url.PathEscape(id), nil, &raw); err != nil {
		return Check{}, err
	}
	check, err := raw.normalize()
	if err != nil || check.ID != id {
		return Check{}, ErrInvalidResponse
	}
	return check, nil
}

// RunChecks asks the checker for an extra run of every target (POST /v1/checks/run {}).
// The response body is not part of the agreed contract yet, so it is ignored.
func (c *Client) RunChecks(ctx context.Context) error {
	return c.do(ctx, http.MethodPost, "/v1/checks/run", struct{}{}, nil)
}

func (c *Client) do(ctx context.Context, method, path string, in, out any) error {
	if c.base == "" {
		return ErrNotConfigured
	}
	var reader io.Reader
	if in != nil {
		raw, err := json.Marshal(in)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, reader)
	if err != nil {
		return ErrNotConfigured
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	req.Header.Set("Accept", "application/json")
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return ErrUnavailable
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusNotFound:
		return ErrNotFound
	case resp.StatusCode == http.StatusConflict:
		return ErrRunning
	case resp.StatusCode < 200 || resp.StatusCode > 299:
		return ErrUnavailable
	}
	if out == nil {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxResponseBytes))
		return nil
	}
	decoder := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err := decoder.Decode(out); err != nil {
		return ErrInvalidResponse
	}
	var trailing any
	if decoder.Decode(&trailing) != io.EOF {
		return ErrInvalidResponse
	}
	return nil
}
