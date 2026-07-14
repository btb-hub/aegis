package resolve

import (
	"encoding/json"
	"strings"
)

const (
	ReasonOK               = ""
	ReasonSlotDisabled     = "slot_disabled"
	ReasonSlotMissing      = "slot_missing"
	ReasonCustomIncomplete = "custom_incomplete"
	ReasonNoGlobal         = "no_global"
	ReasonGlobalDisabled   = "global_disabled"
)

type Input struct {
	Kind   string
	Slot   *Slot
	Global *Slot
}

type Slot struct {
	Mode    string
	Enabled bool
	Config  []byte
}

type Result struct {
	OK     bool
	Config []byte
	Reason string
}

func Resolve(in Input) Result {
	if in.Slot == nil {
		return Result{Reason: ReasonSlotMissing}
	}
	if !in.Slot.Enabled {
		return Result{Reason: ReasonSlotDisabled}
	}

	if in.Slot.Mode == "custom" {
		if !ConfigComplete(in.Kind, in.Slot.Config) {
			return Result{Reason: ReasonCustomIncomplete}
		}
		return Result{OK: true, Config: in.Slot.Config}
	}

	if in.Global == nil {
		return Result{Reason: ReasonNoGlobal}
	}
	if !in.Global.Enabled {
		return Result{Reason: ReasonGlobalDisabled}
	}
	merged, err := MergeConfig(in.Global.Config, in.Slot.Config)
	if err != nil || !ConfigComplete(in.Kind, merged) {
		return Result{Reason: ReasonCustomIncomplete}
	}
	return Result{OK: true, Config: merged}
}

func ConfigComplete(kind string, raw []byte) bool {
	switch kind {
	case "jira":
		var cfg struct {
			BaseURL    string `json:"base_url"`
			Email      string `json:"email"`
			APIToken   string `json:"api_token"`
			ProjectKey string `json:"project_key"`
		}
		return json.Unmarshal(raw, &cfg) == nil &&
			nonBlank(cfg.BaseURL, cfg.Email, cfg.APIToken, cfg.ProjectKey)
	case "slack":
		var cfg struct {
			BotToken      string `json:"bot_token"`
			SigningSecret string `json:"signing_secret"`
		}
		return json.Unmarshal(raw, &cfg) == nil &&
			nonBlank(cfg.BotToken, cfg.SigningSecret)
	case "express":
		var cfg struct {
			BotID     string `json:"bot_id"`
			Host      string `json:"host"`
			SecretKey string `json:"secret_key"`
		}
		return json.Unmarshal(raw, &cfg) == nil &&
			nonBlank(cfg.BotID, cfg.Host, cfg.SecretKey)
	default:
		return false
	}
}

func MergeConfig(global, overlay []byte) ([]byte, error) {
	var globalMap map[string]any
	if err := json.Unmarshal(global, &globalMap); err != nil {
		return nil, err
	}
	var overlayMap map[string]any
	if err := json.Unmarshal(overlay, &overlayMap); err != nil {
		return nil, err
	}
	if globalMap == nil {
		globalMap = make(map[string]any)
	}
	for key, value := range overlayMap {
		globalMap[key] = value
	}
	return json.Marshal(globalMap)
}

func nonBlank(values ...string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return false
		}
	}
	return true
}
