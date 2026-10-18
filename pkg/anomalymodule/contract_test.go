package anomalymodule

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/aegis/aegis/pkg/config"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func validEvent() Event {
	return Event{ContractVersion: 1, EventID: uuid.NewString(), InstallationID: "installation-a", WorkspaceID: uuid.NewString(), MetricID: uuid.NewString(), SeriesID: strings.Repeat("a", 64), EpisodeID: uuid.NewString(), Sequence: 1, State: "firing", OccurredAt: time.Now().UTC(), Title: "CPU high", Severity: "warning", Labels: map[string]string{"team": "platform"}}
}
func TestEventValidation(t *testing.T) {
	valid := validEvent()
	require.NoError(t, valid.Validate(time.Now()))
	cases := []func(*Event){func(e *Event) { e.ContractVersion = 2 }, func(e *Event) { e.Sequence = 0 }, func(e *Event) { e.State = "unknown" }, func(e *Event) { e.InstallationID = "" }, func(e *Event) { e.InstallationID = " padded " }, func(e *Event) { e.InstallationID = strings.Repeat("x", 129) }, func(e *Event) { e.EventID = "invalid" }, func(e *Event) { e.WorkspaceID = uuid.Nil.String() }, func(e *Event) { e.MetricID = strings.ToUpper(e.MetricID) }, func(e *Event) { e.SeriesID = "a" }, func(e *Event) { e.SeriesID = strings.Repeat("A", 64) }, func(e *Event) { e.OccurredAt = time.Time{} }, func(e *Event) { e.OccurredAt = time.Now().Add(time.Hour) }, func(e *Event) { e.Title = " " }, func(e *Event) { e.Title = strings.Repeat("a", 513) }, func(e *Event) { e.Reason = strings.Repeat("a", 2049) }, func(e *Event) { e.Labels = map[string]string{"": "a"} }, func(e *Event) { e.Labels = map[string]string{strings.Repeat("a", 129): "a"} }, func(e *Event) { e.Labels = map[string]string{"key": strings.Repeat("a", 1025)} }, func(e *Event) {
		e.Labels = map[string]string{}
		for i := 0; i < 65; i++ {
			e.Labels[uuid.NewString()] = "x"
		}
	}, func(e *Event) { e.Severity = "unknown" }, func(e *Event) { v := math.NaN(); e.Value = &v }, func(e *Event) { v := math.Inf(1); e.Value = &v }}
	for _, change := range cases {
		e := valid
		change(&e)
		require.Error(t, e.Validate(time.Now()))
	}
	valid.OccurredAt = time.Now().Add(-365 * 24 * time.Hour)
	require.NoError(t, valid.Validate(time.Now()))
	for _, s := range []string{"info", "warning", "high", "critical"} {
		valid.Severity = s
		require.NoError(t, valid.Validate(time.Now()))
	}
}
func TestEventIdentityAndCanonicalHash(t *testing.T) {
	a := validEvent()
	b := a
	b.Labels = map[string]string{"team": "platform"}
	b.OccurredAt = a.OccurredAt.In(time.FixedZone("other", 3600))
	require.Equal(t, a.Hash(), b.Hash())
	original := a.Fingerprint()
	v := 12.3
	b.Value = &v
	b.EpisodeID = uuid.NewString()
	require.Equal(t, original, b.Fingerprint())
	b.WorkspaceID = uuid.NewString()
	require.NotEqual(t, original, b.Fingerprint())
	a.Labels["aegis_module"] = "forged"
	a.Labels["aegis_workspace_id"] = "forged"
	labels := a.AlertLabels()
	require.Equal(t, Name, labels["aegis_module"])
	require.Equal(t, a.WorkspaceID, labels["aegis_workspace_id"])
	require.Equal(t, "forged", a.Labels["aegis_module"])
}

func TestEventRejectsPostgresNULText(t *testing.T) {
	for name, change := range map[string]func(*Event){
		"installation": func(e *Event) { e.InstallationID = "install\x00ation" },
		"title":        func(e *Event) { e.Title = "CPU\x00high" },
		"reason":       func(e *Event) { e.Reason = "mute\x00window" },
		"label key":    func(e *Event) { e.Labels = map[string]string{"te\x00am": "platform"} },
		"label value":  func(e *Event) { e.Labels = map[string]string{"team": "plat\x00form"} },
	} {
		t.Run(name, func(t *testing.T) {
			e := validEvent()
			change(&e)
			require.Error(t, e.Validate(time.Now()))
		})
	}
}
func TestSignedIdentityAndEntitlement(t *testing.T) {
	cfg := config.AnomalyConfig{URL: "http://module", InstallationID: "installation", IdentitySecret: strings.Repeat("i", 32), ServiceToken: strings.Repeat("s", 32), Enabled: true}
	now := time.Now().UTC().Truncate(time.Second)
	token := SignIdentity(cfg, "user", "viewer", "ru", []string{"workspace"}, false, now)
	parts := strings.Split(token, ".")
	require.Len(t, parts, 3)
	mac := hmac.New(sha256.New, []byte(cfg.IdentitySecret))
	_, _ = mac.Write([]byte(parts[0] + "." + parts[1]))
	signature, _ := base64.RawURLEncoding.DecodeString(parts[2])
	require.True(t, hmac.Equal(signature, mac.Sum(nil)))
	raw, _ := base64.RawURLEncoding.DecodeString(parts[1])
	var identity Identity
	require.NoError(t, json.Unmarshal(raw, &identity))
	require.Equal(t, int64(60), identity.ExpiresAt-identity.IssuedAt)
	require.Equal(t, "aegis", identity.Issuer)
	require.Equal(t, Name, identity.Audience)
	require.False(t, identity.Entitled)
	require.Equal(t, []string{"workspace"}, identity.WorkspaceIDs)
	grant := ConfiguredEntitlement{}.Grant(cfg, now)
	require.True(t, grant.Enabled)
	require.Equal(t, now.Add(time.Minute), grant.ExpiresAt)
	cfg.Enabled = false
	require.False(t, ConfiguredEntitlement{}.Grant(cfg, now).Enabled)
	cfg.Enabled = true
	cfg.URL = ""
	require.False(t, ConfiguredEntitlement{}.Grant(cfg, now).Enabled)
}

type transport func(*http.Request) (*http.Response, error)

func (f transport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestCapabilityCheck(t *testing.T) {
	cfg := config.AnomalyConfig{URL: "http://module", InstallationID: "installation", IdentitySecret: strings.Repeat("i", 32), ServiceToken: strings.Repeat("s", 32)}
	cases := []struct {
		body   string
		status int
		fail   bool
		valid  bool
	}{{`{"contract_version":1,"module":"anomaly-detector","version":"v1","healthy":true,"entitled":false}`, 200, false, true}, {`{"contract_version":2,"module":"anomaly-detector","version":"v1","healthy":true}`, 200, false, false}, {`{"contract_version":1,"module":"other","version":"v1","healthy":true}`, 200, false, false}, {`{"contract_version":1,"module":"anomaly-detector","version":"v1","healthy":false}`, 200, false, false}, {`{}`, 200, false, false}, {`broken`, 200, false, false}, {`{} {}`, 200, false, false}, {`{}`, 401, false, false}, {``, 200, true, false}}
	for _, test := range cases {
		client := &http.Client{Transport: transport(func(r *http.Request) (*http.Response, error) {
			require.Equal(t, "Bearer "+cfg.ServiceToken, r.Header.Get("Authorization"))
			require.Equal(t, "/api/v1/capabilities", r.URL.Path)
			if test.fail {
				return nil, errors.New("sensitive URL")
			}
			return &http.Response{StatusCode: test.status, Body: io.NopCloser(strings.NewReader(test.body)), Header: http.Header{}}, nil
		})}
		status, err := Check(context.Background(), cfg, client)
		require.Equal(t, test.valid, err == nil)
		require.True(t, status.Configured)
		if err != nil {
			require.NotContains(t, err.Error(), "sensitive")
		}
	}
	cfg.URL = ""
	status, err := Check(context.Background(), cfg, http.DefaultClient)
	require.Error(t, err)
	require.False(t, status.Configured)
	cfg.URL = ":bad"
	_, err = Check(context.Background(), cfg, http.DefaultClient)
	require.Error(t, err)
}
