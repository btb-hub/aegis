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
	"strings"
	"time"
	"unicode"

	"github.com/aegis/aegis/pkg/config"
)

const (
	StatusRunning = "running"
	StatusPassed  = "passed"
	StatusFailed  = "failed"

	maxResponseBytes = 1 << 20
	maxChecks        = 1000
)

var (
	ErrNotConfigured   = errors.New("backup checker is not configured")
	ErrUnavailable     = errors.New("backup checker is unavailable")
	ErrInvalidResponse = errors.New("backup checker returned an invalid response")
	ErrNotFound        = errors.New("backup check not found")
	ErrRunning         = errors.New("a check for this backup is already running")

	backupName = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,127}$`)
)

// Check is one backup-checker run.
type Check struct {
	ID                  string     `json:"id"`
	Backup              string     `json:"backup"`
	Status              string     `json:"status"`
	Trigger             string     `json:"trigger"`
	StartedAt           time.Time  `json:"started_at"`
	FinishedAt          *time.Time `json:"finished_at"`
	ReplicasMatch       *bool      `json:"replicas_match"`
	RestoreOK           *bool      `json:"restore_ok"`
	ManifestCountsMatch *bool      `json:"manifest_counts_match"`
	Message             string     `json:"message,omitempty"`
}

// Finished reports whether the run has a final result.
func (c Check) Finished() bool { return c.Status == StatusPassed || c.Status == StatusFailed }

// ValidBackupName reports whether name follows the checker's backup naming rule.
func ValidBackupName(name string) bool { return backupName.MatchString(name) }

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

func (c Check) validate() error {
	switch {
	case !ValidCheckID(c.ID):
		return fmt.Errorf("invalid check id")
	case !ValidBackupName(c.Backup):
		return fmt.Errorf("invalid backup name")
	case c.Status != StatusRunning && !c.Finished():
		return fmt.Errorf("invalid check status")
	case c.Trigger != "" && c.Trigger != "scheduled" && c.Trigger != "manual":
		return fmt.Errorf("invalid check trigger")
	case c.StartedAt.IsZero():
		return fmt.Errorf("missing started_at")
	case c.Finished() && c.FinishedAt == nil:
		return fmt.Errorf("missing finished_at")
	case len(c.Message) > 2000:
		return fmt.Errorf("message too long")
	}
	return nil
}

// Client calls the backup-checker HTTP API with a bearer token.
type Client struct {
	base  string
	token string
	http  *http.Client
}

// New returns a client for cfg. A nil httpClient uses a 10 second timeout and
// never follows redirects, so the bearer token only reaches the configured host.
func New(cfg config.BackupConfig, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	return &Client{base: strings.TrimRight(cfg.CheckerURL, "/"), token: cfg.CheckerToken, http: httpClient}
}

// ListChecks returns recent checks, newest first.
func (c *Client) ListChecks(ctx context.Context) ([]Check, error) {
	var body struct {
		Items []Check `json:"items"`
	}
	if err := c.do(ctx, http.MethodGet, "/v1/checks", nil, &body); err != nil {
		return nil, err
	}
	if body.Items == nil || len(body.Items) > maxChecks {
		return nil, ErrInvalidResponse
	}
	for _, check := range body.Items {
		if check.validate() != nil {
			return nil, ErrInvalidResponse
		}
	}
	return body.Items, nil
}

// GetCheck returns one check run.
func (c *Client) GetCheck(ctx context.Context, id string) (Check, error) {
	if !ValidCheckID(id) {
		return Check{}, ErrNotFound
	}
	var check Check
	if err := c.do(ctx, http.MethodGet, "/v1/checks/"+url.PathEscape(id), nil, &check); err != nil {
		return Check{}, err
	}
	if check.validate() != nil || check.ID != id {
		return Check{}, ErrInvalidResponse
	}
	return check, nil
}

// RunCheck asks the checker for an extra check of one backup.
func (c *Client) RunCheck(ctx context.Context, backup string) (Check, error) {
	if !ValidBackupName(backup) {
		return Check{}, ErrNotFound
	}
	var check Check
	if err := c.do(ctx, http.MethodPost, "/v1/checks/run", map[string]string{"backup": backup}, &check); err != nil {
		return Check{}, err
	}
	if check.validate() != nil || check.Backup != backup {
		return Check{}, ErrInvalidResponse
	}
	return check, nil
}

func (c *Client) do(ctx context.Context, method, path string, in, out any) error {
	if c.base == "" || c.token == "" {
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
	req.Header.Set("Authorization", "Bearer "+c.token)
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
