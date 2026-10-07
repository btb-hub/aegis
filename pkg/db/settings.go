package db

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"

	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/aegis/aegis/pkg/config"
	"github.com/aegis/aegis/pkg/sessiontoken"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const settingsColumns = `values, enabled, revision, imported, bootstrap_closed`

func scanSettings(row pgx.Row) (config.Settings, error) {
	var s config.Settings
	var values, enabled []byte
	if err := row.Scan(&values, &enabled, &s.Revision, &s.Imported, &s.BootstrapClosed); err != nil {
		return s, err
	}
	if err := json.Unmarshal(values, &s.Values); err != nil {
		return s, err
	}
	err := json.Unmarshal(enabled, &s.Enabled)
	return s, err
}

func (s *Store) GetSettings(ctx context.Context) (config.Settings, error) {
	if snapshot, ok := config.SettingsSnapshot(ctx); ok {
		return snapshot, nil
	}
	return scanSettings(s.pool.QueryRow(ctx, `SELECT `+settingsColumns+` FROM application_settings WHERE singleton`))
}

// SettingsContext is shared by API requests, worker jobs, and chat delivery tasks.
func (s *Store) SettingsContext(ctx context.Context, boot *config.Config) (context.Context, error) {
	doc, err := s.GetSettings(ctx)
	if err != nil {
		return ctx, err
	}
	runtime, err := doc.Config(boot)
	if err != nil {
		return ctx, err
	}
	return config.WithSettings(ctx, doc, runtime), nil
}

func (s *Store) SettingsConfig(ctx context.Context, boot *config.Config) (*config.Config, error) {
	doc, err := s.GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	return doc.Config(boot)
}

func lockSettings(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(170122, 1)`)
	return err
}

func writeSettings(ctx context.Context, tx pgx.Tx, doc config.Settings) error {
	values, err := json.Marshal(doc.Values)
	if err != nil {
		return err
	}
	enabled, err := json.Marshal(doc.Enabled)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO application_settings(singleton,values,enabled,revision,imported,bootstrap_closed)
 VALUES(true,$1,$2,$3,$4,$5) ON CONFLICT(singleton) DO UPDATE SET values=EXCLUDED.values,enabled=EXCLUDED.enabled,
 revision=EXCLUDED.revision, imported=EXCLUDED.imported,bootstrap_closed=EXCLUDED.bootstrap_closed,updated_at=now()`, values, enabled, doc.Revision, doc.Imported, doc.BootstrapClosed)
	return err
}

func settingsAudit(ctx context.Context, tx pgx.Tx, actor *uuid.UUID, action, section string, revision int64, keys []string) error {
	if keys == nil {
		keys = []string{}
	}
	details, _ := json.Marshal(map[string]any{"section": section, "revision": revision, "fields": keys})
	_, err := tx.Exec(ctx, `INSERT INTO audit_log(actor_id,action,resource_type,resource_id,details) VALUES($1,$2,'settings',$3,$4)`, actor, action, uuid.Nil, details)
	return err
}

// InitializeSettings never imports env on startup. Upgrades require the explicit importer.
func (s *Store) InitializeSettings(ctx context.Context, boot *config.Config) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err = lockSettings(ctx, tx); err != nil {
		return err
	}
	doc, err := scanSettings(tx.QueryRow(ctx, `SELECT `+settingsColumns+` FROM application_settings WHERE singleton`))
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	var existing bool
	if e := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users) OR EXISTS(SELECT 1 FROM teams) OR EXISTS(SELECT 1 FROM integrations WHERE workspace_id IS NULL OR config<>'{}'::jsonb) OR EXISTS(SELECT 1 FROM alerts) OR EXISTS(SELECT 1 FROM incidents) OR EXISTS(SELECT 1 FROM workspaces WHERE slug<>'default')`).Scan(&existing); e != nil {
		return e
	}
	if errors.Is(err, pgx.ErrNoRows) {
		if existing {
			return errors.New("configuration import required: run config-import before starting updated services")
		}
		doc = config.DefaultSettings()
		webhook, _, e := sessiontoken.New()
		if e != nil {
			return e
		}
		doc.Values["WEBHOOK_SECRET"] = webhook
		if boot.DevAuthEnabled {
			doc.Imported = true
			doc.BootstrapClosed = true
		}
		if err = writeSettings(ctx, tx, doc); err != nil {
			return err
		}
	} else if existing && !doc.Imported {
		return errors.New("configuration import required")
	}
	return tx.Commit(ctx)
}

func (s *Store) PatchSettings(ctx context.Context, actor uuid.UUID, revision int64, section string, patch map[string]string, confirmURL bool) (config.Settings, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return config.Settings{}, err
	}
	defer tx.Rollback(ctx)
	if err = lockSettings(ctx, tx); err != nil {
		return config.Settings{}, err
	}
	doc, err := scanSettings(tx.QueryRow(ctx, `SELECT `+settingsColumns+` FROM application_settings WHERE singleton FOR UPDATE`))
	if err != nil {
		return doc, err
	}
	if doc.Revision != revision {
		return doc, apperrors.Conflict("settings changed; reload before saving")
	}
	allowed := map[string]bool{}
	for _, key := range config.SettingSections[section] {
		allowed[key] = true
	}
	if len(allowed) == 0 || section == "compatibility" {
		return doc, apperrors.Validation("unknown settings section", nil)
	}
	changed := []string{}
	previousURL := doc.Values["PUBLIC_URL"]
	for key, value := range patch {
		if !allowed[key] {
			return doc, apperrors.Validation("unsupported setting", map[string]any{"key": key})
		}
		if config.SecretKey(key) && value == "" {
			continue
		}
		if value != doc.Values[key] {
			doc.Values[key] = value
			changed = append(changed, key)
		}
	}
	if err = config.ValidateValues(doc.Values); err != nil {
		return doc, apperrors.Validation(err.Error(), nil)
	}
	if runtime := config.Runtime(ctx, nil); runtime != nil && runtime.DevAuthEnabled {
		if _, err = doc.Config(runtime); err != nil {
			return doc, apperrors.Validation("invalid setting PUBLIC_URL while development sign-in is enabled", nil)
		}
	}
	urlChanged := false
	for _, key := range changed {
		if key == "PUBLIC_URL" {
			urlChanged = true
		}
	}
	if urlChanged && !confirmURL {
		return doc, apperrors.Conflict("confirm the new provider callback URLs before saving")
	}
	if urlChanged {
		for _, provider := range []string{"google", "slack", "express"} {
			key := strings.ToUpper(provider) + "_OIDC_REDIRECT_URL"
			oldCallback := strings.TrimRight(previousURL, "/") + "/auth/" + provider + "/callback"
			newCallback := strings.TrimRight(doc.Values["PUBLIC_URL"], "/") + "/auth/" + provider + "/callback"
			if doc.Values[key] == oldCallback {
				doc.Values[key] = newCallback
				changed = append(changed, key)
			}
			if _, err = tx.Exec(ctx, `UPDATE settings_provider_drafts SET values=jsonb_set(values,ARRAY[$1],to_jsonb($2::text)),revision=revision+1 WHERE provider=$3 AND values->>$1=$4`, key, newCallback, provider, oldCallback); err != nil {
				return doc, err
			}
		}
		if _, err = tx.Exec(ctx, `DELETE FROM settings_authorizations`); err != nil {
			return doc, err
		}
		if _, err = tx.Exec(ctx, `DELETE FROM paging_authorizations`); err != nil {
			return doc, err
		}
		if _, err = tx.Exec(ctx, `UPDATE settings_provider_drafts SET tested_revision=NULL,tested_by=NULL,tested_at=NULL`); err != nil {
			return doc, err
		}
	}
	if len(changed) > 0 {
		doc.Revision++
		sort.Strings(changed)
		if err = writeSettings(ctx, tx, doc); err != nil {
			return doc, err
		}
		if err = settingsAudit(ctx, tx, &actor, "settings.updated", section, doc.Revision, changed); err != nil {
			return doc, err
		}
	}
	err = tx.Commit(ctx)
	return doc, err
}

type ImportStatus struct {
	Key    string `json:"key"`
	Status string `json:"status"`
}

// ImportEnvironment serializes imports and all settings writers. Dry runs roll back.
func (s *Store) ImportEnvironment(ctx context.Context, env map[string]string, apply bool) ([]ImportStatus, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, errors.New("cannot connect to configuration database")
	}
	defer tx.Rollback(ctx)
	if err = lockSettings(ctx, tx); err != nil {
		return nil, errors.New("cannot lock configuration database")
	}
	doc, err := scanSettings(tx.QueryRow(ctx, `SELECT `+settingsColumns+` FROM application_settings WHERE singleton FOR UPDATE`))
	absent := errors.Is(err, pgx.ErrNoRows)
	if err != nil && !absent {
		return nil, errors.New("cannot read stored settings")
	}
	if absent {
		doc = config.Settings{Values: map[string]string{}, Enabled: map[string]bool{}, Revision: 1}
	}
	report := []ImportStatus{}
	blocked := false
	changed := []string{}
	keys := make([]string, 0, len(env))
	for key := range env {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	connectorEnv := map[string]map[string]string{}
	for _, key := range keys {
		value := env[key]
		status := "importable"
		if !config.KnownKey(key) {
			status = "unsupported"
			blocked = true
		} else if config.DeploymentKey(key) {
			status = "deployment-only"
		} else {
			connector := ""
			field := ""
			for kind, fields := range config.ConnectorKeys {
				if f, ok := fields[key]; ok {
					connector = kind
					field = f
					break
				}
			}
			if connector != "" {
				if connectorEnv[connector] == nil {
					connectorEnv[connector] = map[string]string{}
				}
				connectorEnv[connector][field] = value
			} else if _, exists := doc.Values[key]; exists {
				status = "already stored"
			} else {
				doc.Values[key] = value
				changed = append(changed, key)
			}
		}
		report = append(report, ImportStatus{key, status})
	}
	defaults := config.DefaultSettings()
	for key, value := range defaults.Values {
		if _, ok := doc.Values[key]; !ok {
			doc.Values[key] = value
		}
	}
	if strings.TrimSpace(doc.Values["WEBHOOK_SECRET"]) == "" {
		return report, errors.New("invalid setting WEBHOOK_SECRET")
	}
	if err = config.ValidateValues(doc.Values); err != nil {
		return report, err
	}
	for _, p := range []string{"google", "slack", "express"} {
		if _, exists := doc.Enabled[p]; !exists {
			prefix := strings.ToUpper(p) + "_OIDC_"
			complete := true
			any := false
			for _, key := range config.ProviderKeys(p) {
				if doc.Values[key] == "" {
					complete = false
				} else {
					any = true
				}
			}
			doc.Enabled[p] = complete
			if any && !complete {
				for i := range report {
					if strings.HasPrefix(report[i].Key, prefix) && report[i].Status == "importable" {
						report[i].Status = "incomplete"
					}
				}
			}
		}
	}
	for kind, fields := range connectorEnv {
		var id uuid.UUID
		var raw []byte
		var enabled bool
		err = tx.QueryRow(ctx, `SELECT id,config,enabled FROM integrations WHERE kind=$1 AND workspace_id IS NULL FOR UPDATE`, kind).Scan(&id, &raw, &enabled)
		isNew := errors.Is(err, pgx.ErrNoRows)
		if err != nil && !isNew {
			return report, errors.New("cannot read connector settings")
		}
		stored := map[string]any{}
		if !isNew {
			if json.Unmarshal(raw, &stored) != nil {
				return report, errors.New("invalid stored connector settings")
			}
		}
		connectorChanged := isNew
		for key, field := range config.ConnectorKeys[kind] {
			v, present := fields[field]
			if !present {
				continue
			}
			status := "importable"
			if existing, ok := stored[field]; ok && existing != nil && existing != "" {
				status = "already stored"
			} else if v != "" {
				stored[field] = v
				connectorChanged = true
				changed = append(changed, key)
			} else if isNew {
				changed = append(changed, key)
			}
			for i := range report {
				if report[i].Key == key {
					report[i].Status = status
				}
			}
		}
		complete := connectorComplete(kind, stored)
		for key, field := range config.ConnectorKeys[kind] {
			if value, ok := stored[field].(string); ok {
				if err = config.ValidateValues(map[string]string{key: value}); err != nil {
					return report, err
				}
			}
		}
		if isNew && !complete {
			for i := range report {
				if _, ok := config.ConnectorKeys[kind][report[i].Key]; ok && report[i].Status == "importable" {
					report[i].Status = "incomplete"
				}
			}
		}
		if apply && connectorChanged {
			payload, _ := json.Marshal(stored)
			if isNew {
				_, err = tx.Exec(ctx, `INSERT INTO integrations(kind,name,config,enabled) VALUES($1,$1,$2,$3)`, kind, payload, complete)
			} else {
				_, err = tx.Exec(ctx, `UPDATE integrations SET config=$2,updated_at=now() WHERE id=$1`, id, payload)
			}
			if err != nil {
				return report, errors.New("cannot import connector settings")
			}
		}
	}
	if blocked {
		return report, errors.New("unsupported keys must be accounted for before import")
	}
	if !apply {
		return report, nil
	}
	if doc.Imported && len(changed) == 0 {
		return report, nil
	}
	doc.Imported = true
	doc.BootstrapClosed = true
	if !absent {
		doc.Revision++
	}
	if err = writeSettings(ctx, tx, doc); err != nil {
		return report, errors.New("cannot import application settings")
	}
	if err = settingsAudit(ctx, tx, nil, "settings.imported", "environment", doc.Revision, changed); err != nil {
		return report, errors.New("cannot write import audit")
	}
	if err = tx.Commit(ctx); err != nil {
		return report, errors.New("cannot commit configuration import")
	}
	return report, nil
}

func connectorComplete(kind string, fields map[string]any) bool {
	required := map[string][]string{"jira": {"base_url", "api_token", "project_key"}, "slack": {"bot_token", "signing_secret"}, "express": {"bot_id", "host", "secret_key"}}
	for _, key := range required[kind] {
		v, ok := fields[key].(string)
		if !ok || strings.TrimSpace(v) == "" {
			return false
		}
	}
	return true
}

func settingsProvider(provider string) bool {
	return provider == "google" || provider == "slack" || provider == "express"
}
func settingConflict() error {
	return apperrors.Conflict("authorization or settings changed; try again")
}
func settingInvalid() error { return apperrors.Validation("authorization invalid or expired", nil) }
