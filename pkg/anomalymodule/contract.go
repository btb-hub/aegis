package anomalymodule

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/aegis/aegis/pkg/config"
	"github.com/google/uuid"
)

const Name = "anomaly-detector"
const ContractVersion = 1
const Prefix = "/modules/anomaly-detector"

type Event struct {
	ContractVersion int               `json:"contract_version"`
	EventID         string            `json:"event_id"`
	InstallationID  string            `json:"installation_id"`
	WorkspaceID     string            `json:"workspace_id"`
	MetricID        string            `json:"metric_id"`
	SeriesID        string            `json:"series_id"`
	EpisodeID       string            `json:"episode_id"`
	Sequence        int64             `json:"sequence"`
	State           string            `json:"state"`
	OccurredAt      time.Time         `json:"occurred_at"`
	Title           string            `json:"title"`
	Severity        string            `json:"severity"`
	Labels          map[string]string `json:"labels,omitempty"`
	Value           *float64          `json:"value,omitempty"`
	Reason          string            `json:"reason,omitempty"`
}

func (e Event) Validate(now time.Time) error {
	if e.ContractVersion != ContractVersion || e.Sequence < 1 || (e.State != "firing" && e.State != "resolved") {
		return fmt.Errorf("invalid event contract, sequence or state")
	}
	if len(e.InstallationID) == 0 || len(e.InstallationID) > 128 || strings.TrimSpace(e.InstallationID) != e.InstallationID || strings.ContainsRune(e.InstallationID, 0) {
		return fmt.Errorf("invalid installation identity")
	}
	for _, value := range []string{e.EventID, e.WorkspaceID, e.MetricID, e.EpisodeID} {
		id, err := uuid.Parse(value)
		if err != nil || id == uuid.Nil || id.String() != value {
			return fmt.Errorf("event identities must be canonical UUIDs")
		}
	}
	raw, err := hex.DecodeString(e.SeriesID)
	if err != nil || len(raw) != sha256.Size || strings.ToLower(e.SeriesID) != e.SeriesID {
		return fmt.Errorf("series_id must be a lowercase SHA256 digest")
	}
	if e.OccurredAt.IsZero() || e.OccurredAt.After(now.Add(5*time.Minute)) {
		return fmt.Errorf("invalid event timestamp")
	}
	if len(strings.TrimSpace(e.Title)) == 0 || len(e.Title) > 512 || len(e.Reason) > 2048 || len(e.Labels) > 64 || strings.ContainsRune(e.Title, 0) || strings.ContainsRune(e.Reason, 0) {
		return fmt.Errorf("invalid event text or labels")
	}
	for k, v := range e.Labels {
		if k == "" || len(k) > 128 || len(v) > 1024 || strings.ContainsRune(k, 0) || strings.ContainsRune(v, 0) {
			return fmt.Errorf("invalid event label")
		}
	}
	switch e.Severity {
	case "info", "warning", "high", "critical":
	default:
		return fmt.Errorf("invalid event severity")
	}
	if e.Value != nil && (math.IsNaN(*e.Value) || math.IsInf(*e.Value, 0)) {
		return fmt.Errorf("event value must be finite")
	}
	return nil
}

func (e Event) Hash() string {
	e.OccurredAt = e.OccurredAt.UTC()
	raw, _ := json.Marshal(e)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
func (e Event) Fingerprint() string {
	raw, _ := json.Marshal([]string{e.InstallationID, e.WorkspaceID, e.MetricID, e.SeriesID})
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
func (e Event) AlertLabels() map[string]string {
	labels := make(map[string]string, len(e.Labels)+5)
	for k, v := range e.Labels {
		labels[k] = v
	}
	labels["aegis_module"] = Name
	labels["aegis_workspace_id"] = e.WorkspaceID
	labels["aegis_installation_id"] = e.InstallationID
	labels["aegis_metric_id"] = e.MetricID
	labels["aegis_series_id"] = e.SeriesID
	return labels
}

type Identity struct {
	Issuer         string   `json:"iss"`
	Audience       string   `json:"aud"`
	Subject        string   `json:"sub"`
	InstallationID string   `json:"installation_id"`
	Role           string   `json:"role"`
	Locale         string   `json:"locale"`
	WorkspaceIDs   []string `json:"workspace_ids"`
	Entitled       bool     `json:"entitled"`
	IssuedAt       int64    `json:"iat"`
	ExpiresAt      int64    `json:"exp"`
}

func SignIdentity(c config.AnomalyConfig, userID, role, locale string, workspaces []string, entitled bool, now time.Time) string {
	claims := Identity{Issuer: "aegis", Audience: Name, Subject: userID, InstallationID: c.InstallationID, Role: role, Locale: locale, WorkspaceIDs: workspaces, Entitled: entitled, IssuedAt: now.Unix(), ExpiresAt: now.Add(time.Minute).Unix()}
	raw, _ := json.Marshal(claims)
	encoded := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`)) + "." + base64.RawURLEncoding.EncodeToString(raw)
	mac := hmac.New(sha256.New, []byte(c.IdentitySecret))
	_, _ = mac.Write([]byte(encoded))
	return encoded + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

type Grant struct {
	ContractVersion int       `json:"contract_version"`
	InstallationID  string    `json:"installation_id"`
	Enabled         bool      `json:"enabled"`
	ExpiresAt       time.Time `json:"expires_at"`
}

// EntitlementProvider can later validate purchased licenses without changing callers.
type EntitlementProvider interface {
	Grant(config.AnomalyConfig, time.Time) Grant
}
type ConfiguredEntitlement struct{}

func (ConfiguredEntitlement) Grant(c config.AnomalyConfig, now time.Time) Grant {
	return Grant{ContractVersion: ContractVersion, InstallationID: c.InstallationID, Enabled: c.Configured() && c.Enabled, ExpiresAt: now.Add(time.Minute)}
}
