package resolve

import (
	"encoding/json"
	"testing"
)

func TestResolveCustomComplete(t *testing.T) {
	config := []byte(`{"base_url":"https://example.atlassian.net","email":"ops@example.com","api_token":"token","project_key":"OPS"}`)

	got := Resolve(Input{
		Kind: "jira",
		Slot: &Slot{Mode: "custom", Enabled: true, Config: config},
	})

	if !got.OK || got.Reason != ReasonOK || string(got.Config) != string(config) {
		t.Fatalf("Resolve() = %+v, want successful custom config", got)
	}
}

func TestResolveCustomIncomplete(t *testing.T) {
	got := Resolve(Input{
		Kind: "jira",
		Slot: &Slot{
			Mode:    "custom",
			Enabled: true,
			Config:  []byte(`{"base_url":"https://example.atlassian.net","email":"ops@example.com","api_token":"token","project_key":"  "}`),
		},
	})

	if got.OK || got.Reason != ReasonCustomIncomplete {
		t.Fatalf("Resolve() = %+v, want reason %q", got, ReasonCustomIncomplete)
	}
}

func TestResolveInheritMergesProjectKey(t *testing.T) {
	got := Resolve(Input{
		Kind: "jira",
		Slot: &Slot{
			Mode:    "inherit",
			Enabled: true,
			Config:  []byte(`{"project_key":"OPS"}`),
		},
		Global: &Slot{
			Enabled: true,
			Config:  []byte(`{"base_url":"https://example.atlassian.net","email":"ops@example.com","api_token":"token"}`),
		},
	})

	if !got.OK || got.Reason != ReasonOK {
		t.Fatalf("Resolve() = %+v, want successful inherited config", got)
	}
	var config map[string]any
	if err := json.Unmarshal(got.Config, &config); err != nil {
		t.Fatalf("unmarshal resolved config: %v", err)
	}
	for key, want := range map[string]string{
		"base_url":    "https://example.atlassian.net",
		"email":       "ops@example.com",
		"api_token":   "token",
		"project_key": "OPS",
	} {
		if got := config[key]; got != want {
			t.Errorf("config[%q] = %#v, want %q", key, got, want)
		}
	}
}

func TestResolveInheritNoGlobal(t *testing.T) {
	got := Resolve(Input{
		Kind: "jira",
		Slot: &Slot{Mode: "inherit", Enabled: true, Config: []byte(`{"project_key":"OPS"}`)},
	})

	if got.OK || got.Reason != ReasonNoGlobal {
		t.Fatalf("Resolve() = %+v, want reason %q", got, ReasonNoGlobal)
	}
}

func TestResolveSlotDisabled(t *testing.T) {
	got := Resolve(Input{
		Kind: "jira",
		Slot: &Slot{Mode: "custom", Enabled: false},
	})

	if got.OK || got.Reason != ReasonSlotDisabled {
		t.Fatalf("Resolve() = %+v, want reason %q", got, ReasonSlotDisabled)
	}
}

func TestResolveNilSlot(t *testing.T) {
	got := Resolve(Input{Kind: "jira"})

	if got.OK || got.Reason != ReasonSlotMissing {
		t.Fatalf("Resolve() = %+v, want reason %q", got, ReasonSlotMissing)
	}
}

func TestConfigCompleteUsesProviderRequirements(t *testing.T) {
	tests := []struct {
		name string
		kind string
		raw  []byte
		want bool
	}{
		{
			name: "jira",
			kind: "jira",
			raw:  []byte(`{"base_url":"https://example.atlassian.net","email":"ops@example.com","api_token":"token","project_key":"OPS"}`),
			want: true,
		},
		{
			name: "slack",
			kind: "slack",
			raw:  []byte(`{"bot_token":"token","signing_secret":"secret"}`),
			want: true,
		},
		{
			name: "express",
			kind: "express",
			raw:  []byte(`{"bot_id":"bot","host":"https://express.example.com","secret_key":"secret"}`),
			want: true,
		},
		{
			name: "whitespace slack secret",
			kind: "slack",
			raw:  []byte(`{"bot_token":"token","signing_secret":" "}`),
			want: false,
		},
		{
			name: "unknown kind",
			kind: "unknown",
			raw:  []byte(`{}`),
			want: false,
		},
		{
			name: "invalid JSON",
			kind: "jira",
			raw:  []byte(`{`),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ConfigComplete(tt.kind, tt.raw); got != tt.want {
				t.Fatalf("ConfigComplete(%q, %s) = %t, want %t", tt.kind, tt.raw, got, tt.want)
			}
		})
	}
}

func TestMergeConfigOverlayWins(t *testing.T) {
	got, err := MergeConfig(
		[]byte(`{"base_url":"https://global.example.com","project_key":"GLOBAL"}`),
		[]byte(`{"project_key":"OPS"}`),
	)
	if err != nil {
		t.Fatalf("MergeConfig() error = %v", err)
	}

	var config map[string]any
	if err := json.Unmarshal(got, &config); err != nil {
		t.Fatalf("unmarshal merged config: %v", err)
	}
	if config["base_url"] != "https://global.example.com" || config["project_key"] != "OPS" {
		t.Fatalf("MergeConfig() = %v, want global base URL and overlaid project key", config)
	}
}

func TestMergeConfigRejectsInvalidJSON(t *testing.T) {
	if _, err := MergeConfig([]byte(`{`), []byte(`{}`)); err == nil {
		t.Fatal("MergeConfig() error = nil for invalid global JSON")
	}
	if _, err := MergeConfig([]byte(`{}`), []byte(`{`)); err == nil {
		t.Fatal("MergeConfig() error = nil for invalid overlay JSON")
	}
}
