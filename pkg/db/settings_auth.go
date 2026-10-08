package db

import (
	"context"
	"encoding/json"
	"errors"
	"net/mail"
	"strings"
	"time"

	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/aegis/aegis/pkg/config"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type ProviderDraft struct {
	Provider       string            `json:"provider"`
	Values         map[string]string `json:"values"`
	Revision       int64             `json:"revision"`
	TestedRevision *int64            `json:"tested_revision"`
	TestedBy       *string           `json:"-"`
	ExpectedEmail  string            `json:"expected_email"`
}

func scanDraft(row pgx.Row) (ProviderDraft, error) {
	var d ProviderDraft
	var raw []byte
	err := row.Scan(&d.Provider, &raw, &d.Revision, &d.TestedRevision, &d.TestedBy, &d.ExpectedEmail)
	if err == nil {
		err = json.Unmarshal(raw, &d.Values)
	}
	return d, err
}

const draftColumns = `provider, values, revision, tested_revision, tested_by, expected_email`

func (s *Store) GetProviderDraft(ctx context.Context, provider string) (ProviderDraft, error) {
	return scanDraft(s.pool.QueryRow(ctx, `SELECT `+draftColumns+` FROM settings_provider_drafts WHERE provider=$1`, provider))
}

func (s *Store) SaveProviderDraft(ctx context.Context, actor *uuid.UUID, provider string, revision int64, patch map[string]string, email string) (ProviderDraft, error) {
	if !settingsProvider(provider) {
		return ProviderDraft{}, apperrors.Validation("unknown provider", nil)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ProviderDraft{}, err
	}
	defer tx.Rollback(ctx)
	if err = lockSettings(ctx, tx); err != nil {
		return ProviderDraft{}, err
	}
	doc, err := scanSettings(tx.QueryRow(ctx, `SELECT `+settingsColumns+` FROM application_settings WHERE singleton FOR UPDATE`))
	if err != nil {
		return ProviderDraft{}, err
	}
	if actor == nil && (doc.BootstrapClosed || doc.Imported) {
		return ProviderDraft{}, apperrors.NotFound("bootstrap")
	}
	draft, err := scanDraft(tx.QueryRow(ctx, `SELECT `+draftColumns+` FROM settings_provider_drafts WHERE provider=$1 FOR UPDATE`, provider))
	if errors.Is(err, pgx.ErrNoRows) {
		draft = ProviderDraft{Provider: provider, Values: map[string]string{}}
	} else if err != nil {
		return draft, err
	}
	if draft.Revision != revision {
		return draft, settingConflict()
	}
	allowed := map[string]bool{}
	for _, key := range config.ProviderKeys(provider) {
		allowed[key] = true
		if _, exists := draft.Values[key]; !exists {
			draft.Values[key] = doc.Values[key]
		}
	}
	changed := []string{}
	for key, value := range patch {
		if !allowed[key] {
			return draft, apperrors.Validation("unsupported provider setting", map[string]any{"key": key})
		}
		if config.SecretKey(key) && value == "" {
			continue
		}
		draft.Values[key] = value
		changed = append(changed, key)
	}
	redirectKey := strings.ToUpper(provider) + "_OIDC_REDIRECT_URL"
	if draft.Values[redirectKey] == "" {
		draft.Values[redirectKey] = strings.TrimRight(doc.Values["PUBLIC_URL"], "/") + "/auth/" + provider + "/callback"
	}
	if err = config.ValidateValues(draft.Values); err != nil {
		return draft, apperrors.Validation(err.Error(), nil)
	}
	for _, key := range config.ProviderKeys(provider) {
		if strings.TrimSpace(draft.Values[key]) == "" {
			return draft, apperrors.Validation("provider setting is required", map[string]any{"key": key})
		}
	}
	email = strings.ToLower(strings.TrimSpace(email))
	address, emailErr := mail.ParseAddress(email)
	if emailErr != nil || address.Address != email {
		return draft, apperrors.Validation("verified administrator email is required", nil)
	}
	draft.ExpectedEmail = email
	draft.Revision++
	draft.TestedRevision = nil
	draft.TestedBy = nil
	raw, _ := json.Marshal(draft.Values)
	_, err = tx.Exec(ctx, `INSERT INTO settings_provider_drafts(provider,values,revision,expected_email) VALUES($1,$2,$3,$4)
 ON CONFLICT(provider) DO UPDATE SET values=EXCLUDED.values,revision=EXCLUDED.revision,expected_email=EXCLUDED.expected_email,tested_revision=NULL,tested_by=NULL,tested_at=NULL`, provider, raw, draft.Revision, email)
	if err != nil {
		return draft, err
	}
	if err = settingsAudit(ctx, tx, actor, "settings.provider_draft_saved", provider, draft.Revision, changed); err != nil {
		return draft, err
	}
	err = tx.Commit(ctx)
	return draft, err
}

func DraftConfig(doc config.Settings, draft ProviderDraft, boot *config.Config) (*config.Config, error) {
	copyValues := map[string]string{}
	for key, value := range doc.Values {
		copyValues[key] = value
	}
	for key, value := range draft.Values {
		copyValues[key] = value
	}
	doc.Values = copyValues
	doc.Enabled = map[string]bool{draft.Provider: true}
	return doc.Config(boot)
}

func (s *Store) ActivateProvider(ctx context.Context, actor uuid.UUID, sessionHash, provider string, settingsRevision, draftRevision int64, enabled bool) (config.Settings, error) {
	if !settingsProvider(provider) {
		return config.Settings{}, apperrors.Validation("unknown provider", nil)
	}
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
	if doc.Revision != settingsRevision {
		return doc, settingConflict()
	}
	if enabled {
		draft, e := scanDraft(tx.QueryRow(ctx, `SELECT `+draftColumns+` FROM settings_provider_drafts WHERE provider=$1 FOR UPDATE`, provider))
		if e != nil {
			return doc, settingConflict()
		}
		if draft.Revision != draftRevision || draft.TestedRevision == nil || *draft.TestedRevision != draftRevision || draft.TestedBy == nil || *draft.TestedBy != sessionHash {
			return doc, apperrors.Conflict("test this provider draft before activation")
		}
		for key, value := range draft.Values {
			doc.Values[key] = value
		}
	} else {
		remaining := false
		for p, on := range doc.Enabled {
			if p != provider && on {
				remaining = true
			}
		}
		if !remaining {
			return doc, apperrors.Conflict("cannot disable the last active sign-in provider")
		}
	}
	doc.Enabled[provider] = enabled
	doc.Revision++
	if err = writeSettings(ctx, tx, doc); err != nil {
		return doc, err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM settings_authorizations WHERE provider=$1`, provider); err != nil {
		return doc, err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM paging_authorizations WHERE provider=$1`, provider); err != nil {
		return doc, err
	}
	if err = settingsAudit(ctx, tx, &actor, "settings.provider_activated", provider, doc.Revision, []string{"enabled", "credentials"}); err != nil {
		return doc, err
	}
	err = tx.Commit(ctx)
	return doc, err
}

type SettingsAuthorization struct {
	StateHash, Kind, Provider, Nonce, SessionHash, ExpectedEmail string
	DraftRevision, SettingsRevision                              int64
	ExpiresAt                                                    time.Time
}

func (s *Store) CreateSettingsAuthorization(ctx context.Context, a SettingsAuthorization) error {
	result, err := s.pool.Exec(ctx, `INSERT INTO settings_authorizations(state_hash,kind,provider,nonce,session_hash,expected_email,draft_revision,settings_revision,expires_at)
 SELECT $1,$2,$3,$4,$5,$6,$7,$8,$9 FROM application_settings WHERE singleton AND revision=$8`, a.StateHash, a.Kind, a.Provider, a.Nonce, a.SessionHash, a.ExpectedEmail, a.DraftRevision, a.SettingsRevision, a.ExpiresAt)
	if err == nil && result.RowsAffected() != 1 {
		return settingConflict()
	}
	return err
}

func (s *Store) ClaimSettingsAuthorization(ctx context.Context, stateHash, sessionHash, provider string) (SettingsAuthorization, error) {
	var a SettingsAuthorization
	err := s.pool.QueryRow(ctx, `UPDATE settings_authorizations a SET exchanging=true FROM application_settings s
 WHERE a.state_hash=$1 AND a.session_hash=$2 AND a.provider=$3 AND NOT a.exchanging AND a.expires_at>now()
 AND s.singleton AND s.revision=a.settings_revision
 RETURNING a.state_hash,a.kind,a.provider,a.nonce,a.session_hash,a.expected_email,a.draft_revision,a.settings_revision,a.expires_at`, stateHash, sessionHash, provider).Scan(&a.StateHash, &a.Kind, &a.Provider, &a.Nonce, &a.SessionHash, &a.ExpectedEmail, &a.DraftRevision, &a.SettingsRevision, &a.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return a, settingInvalid()
	}
	return a, err
}

func (s *Store) DeleteSettingsAuthorization(ctx context.Context, stateHash string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM settings_authorizations WHERE state_hash=$1`, stateHash)
	return err
}

func (s *Store) ValidateLoginAuthorization(ctx context.Context, a SettingsAuthorization) error {
	var valid bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM settings_authorizations a JOIN application_settings s ON s.singleton
 WHERE a.state_hash=$1 AND a.exchanging AND a.expires_at>now() AND a.settings_revision=s.revision
 AND (s.enabled->>a.provider)::boolean)`, a.StateHash).Scan(&valid)
	if err != nil {
		return err
	}
	if !valid {
		return settingInvalid()
	}
	return nil
}

func validateSettingsAttempt(ctx context.Context, tx pgx.Tx, a SettingsAuthorization, doc config.Settings) error {
	if doc.Revision != a.SettingsRevision {
		return settingConflict()
	}
	var valid bool
	err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM settings_authorizations WHERE state_hash=$1 AND session_hash=$2 AND exchanging AND expires_at>now())`, a.StateHash, a.SessionHash).Scan(&valid)
	if err != nil {
		return err
	}
	if !valid {
		return settingInvalid()
	}
	return nil
}

func (s *Store) FinishProviderTest(ctx context.Context, a SettingsAuthorization) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err = lockSettings(ctx, tx); err != nil {
		return err
	}
	doc, err := scanSettings(tx.QueryRow(ctx, `SELECT `+settingsColumns+` FROM application_settings WHERE singleton FOR UPDATE`))
	if err != nil {
		return err
	}
	if err = validateSettingsAttempt(ctx, tx, a, doc); err != nil {
		return err
	}
	var valid bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM sessions s JOIN users u ON u.id=s.user_id WHERE s.token_hash=$1 AND s.expires_at>now() AND u.role='admin' AND lower(u.email)=$2)`, a.SessionHash, a.ExpectedEmail).Scan(&valid); err != nil {
		return err
	}
	if !valid {
		return settingInvalid()
	}
	result, err := tx.Exec(ctx, `UPDATE settings_provider_drafts SET tested_revision=revision,tested_by=$1,tested_at=now() WHERE provider=$2 AND revision=$3 AND expected_email=$4`, a.SessionHash, a.Provider, a.DraftRevision, a.ExpectedEmail)
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return settingConflict()
	}
	return tx.Commit(ctx)
}

func (s *Store) BootstrapAvailable(ctx context.Context) (bool, error) {
	var available bool
	err := s.pool.QueryRow(ctx, `SELECT NOT bootstrap_closed AND NOT imported AND NOT EXISTS(SELECT 1 FROM users) FROM application_settings WHERE singleton`).Scan(&available)
	return available, err
}

func (s *Store) CreateBootstrapSession(ctx context.Context, hash, publicURL string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err = lockSettings(ctx, tx); err != nil {
		return err
	}
	doc, err := scanSettings(tx.QueryRow(ctx, `SELECT `+settingsColumns+` FROM application_settings WHERE singleton FOR UPDATE`))
	if err != nil {
		return err
	}
	var users bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users)`).Scan(&users); err != nil {
		return err
	}
	if doc.BootstrapClosed || doc.Imported || users {
		return apperrors.NotFound("bootstrap")
	}
	if !config.ValidURL(publicURL) {
		return apperrors.Validation("invalid public URL", nil)
	}
	// A single browser owns installation access until it expires; token submission cannot reset it.
	var active bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM settings_bootstrap_sessions WHERE expires_at>now())`).Scan(&active); err != nil {
		return err
	}
	if active {
		return apperrors.Conflict("installation session already active; use that browser or wait for expiry")
	}
	doc.Values["PUBLIC_URL"] = strings.TrimRight(publicURL, "/")
	doc.Revision++
	if err = writeSettings(ctx, tx, doc); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM settings_authorizations`); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM settings_bootstrap_sessions`); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO settings_bootstrap_sessions(token_hash,expires_at) VALUES($1,now()+interval '30 minutes')`, hash); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) ValidBootstrapSession(ctx context.Context, hash string) (bool, error) {
	var valid bool
	err := s.pool.QueryRow(ctx, `SELECT NOT bootstrap_closed AND NOT imported AND NOT EXISTS(SELECT 1 FROM users) AND EXISTS(SELECT 1 FROM settings_bootstrap_sessions WHERE token_hash=$1 AND expires_at>now()) FROM application_settings WHERE singleton`, hash).Scan(&valid)
	return valid, err
}

func (s *Store) CompleteBootstrap(ctx context.Context, a SettingsAuthorization, info OIDCLoginInput, sessionHash string) (User, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return User{}, err
	}
	defer tx.Rollback(ctx)
	if err = lockSettings(ctx, tx); err != nil {
		return User{}, err
	}
	doc, err := scanSettings(tx.QueryRow(ctx, `SELECT `+settingsColumns+` FROM application_settings WHERE singleton FOR UPDATE`))
	if err != nil {
		return User{}, err
	}
	if err = validateSettingsAttempt(ctx, tx, a, doc); err != nil {
		return User{}, err
	}
	var users, valid bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users),EXISTS(SELECT 1 FROM settings_bootstrap_sessions WHERE token_hash=$1 AND expires_at>now())`, a.SessionHash).Scan(&users, &valid); err != nil {
		return User{}, err
	}
	if users || !valid || doc.BootstrapClosed || doc.Imported || strings.ToLower(strings.TrimSpace(info.Email)) != a.ExpectedEmail {
		return User{}, settingInvalid()
	}
	draft, err := scanDraft(tx.QueryRow(ctx, `SELECT `+draftColumns+` FROM settings_provider_drafts WHERE provider=$1 FOR UPDATE`, a.Provider))
	if err != nil {
		return User{}, err
	}
	if draft.Revision != a.DraftRevision || draft.ExpectedEmail != a.ExpectedEmail {
		return User{}, settingConflict()
	}
	user, err := scanUser(tx.QueryRow(ctx, `INSERT INTO users(provider,provider_sub,email,display_name,role,locale,avatar_url) VALUES($1,$2,$3,$4,'admin','en',NULLIF($5,'')) RETURNING `+userSelectColumns, info.Provider, info.ProviderSub, info.Email, info.DisplayName, info.AvatarURL))
	if err != nil {
		return user, err
	}
	if err = insertIdentityTx(ctx, tx, user.ID, info.Provider, info.ProviderSub); err != nil {
		return user, err
	}
	for key, value := range draft.Values {
		doc.Values[key] = value
	}
	doc.Enabled[a.Provider] = true
	doc.Imported = true
	doc.BootstrapClosed = true
	doc.Revision++
	if err = writeSettings(ctx, tx, doc); err != nil {
		return user, err
	}
	runtime, err := doc.Config(&config.Config{})
	if err != nil {
		return user, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO sessions(user_id,token_hash,expires_at) VALUES($1,$2,$3)`, user.ID, sessionHash, time.Now().Add(runtime.SessionTTL)); err != nil {
		return user, err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM settings_bootstrap_sessions`); err != nil {
		return user, err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM settings_authorizations`); err != nil {
		return user, err
	}
	if err = settingsAudit(ctx, tx, &user.ID, "settings.bootstrap_completed", a.Provider, doc.Revision, []string{"provider", "first_admin"}); err != nil {
		return user, err
	}
	err = tx.Commit(ctx)
	return user, err
}
