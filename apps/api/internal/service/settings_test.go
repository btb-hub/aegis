package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aegis/aegis/apps/api/internal/oidc"
	"github.com/aegis/aegis/pkg/config"
	"github.com/aegis/aegis/pkg/db"
	"github.com/aegis/aegis/pkg/sessiontoken"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type settingsTestRepo struct {
	SettingsRepository
	doc                                                     config.Settings
	draft                                                   db.ProviderDraft
	attempt                                                 db.SettingsAuthorization
	getErr, draftErr, createErr, finishErr, bootstrapErr    error
	available, valid, claimed, deleted, finished, installed bool
}

func (r *settingsTestRepo) GetSettings(context.Context) (config.Settings, error) {
	return r.doc, r.getErr
}
func (r *settingsTestRepo) GetProviderDraft(context.Context, string) (db.ProviderDraft, error) {
	return r.draft, r.draftErr
}
func (r *settingsTestRepo) CreateSettingsAuthorization(_ context.Context, a db.SettingsAuthorization) error {
	r.attempt = a
	r.claimed = false
	r.deleted = false
	return r.createErr
}
func (r *settingsTestRepo) ClaimSettingsAuthorization(_ context.Context, state, session, provider string) (db.SettingsAuthorization, error) {
	if r.claimed || r.deleted || state != r.attempt.StateHash || session != r.attempt.SessionHash || provider != r.attempt.Provider || time.Now().After(r.attempt.ExpiresAt) {
		return db.SettingsAuthorization{}, errors.New("invalid")
	}
	r.claimed = true
	return r.attempt, nil
}
func (r *settingsTestRepo) DeleteSettingsAuthorization(context.Context, string) error {
	r.deleted = true
	return nil
}
func (r *settingsTestRepo) ValidateLoginAuthorization(context.Context, db.SettingsAuthorization) error {
	return r.finishErr
}
func (r *settingsTestRepo) FinishProviderTest(context.Context, db.SettingsAuthorization) error {
	r.finished = r.finishErr == nil
	return r.finishErr
}
func (r *settingsTestRepo) BootstrapAvailable(context.Context) (bool, error) {
	return r.available, r.bootstrapErr
}
func (r *settingsTestRepo) ValidBootstrapSession(context.Context, string) (bool, error) {
	return r.valid, r.bootstrapErr
}
func (r *settingsTestRepo) CreateBootstrapSession(context.Context, string, string) error {
	return r.createErr
}
func (r *settingsTestRepo) CompleteBootstrap(context.Context, db.SettingsAuthorization, db.OIDCLoginInput, string) (db.User, error) {
	r.installed = r.finishErr == nil
	return db.User{}, r.finishErr
}

func settingsServiceFixture() (*SettingsService, *settingsTestRepo, *pagingTestExchange) {
	doc := config.DefaultSettings()
	doc.Enabled["google"] = true
	values := map[string]string{"GOOGLE_OIDC_CLIENT_ID": "client", "GOOGLE_OIDC_CLIENT_SECRET": "private-secret", "GOOGLE_OIDC_REDIRECT_URL": "https://aegis.test/auth/google/callback"}
	for key, value := range values {
		doc.Values[key] = value
	}
	r := &settingsTestRepo{doc: doc, draft: db.ProviderDraft{Provider: "google", Values: values, Revision: 1, ExpectedEmail: "admin@company.com"}, available: true, valid: true}
	x := &pagingTestExchange{info: oidc.PagingIdentity{Subject: "subject", Email: "admin@company.com", EmailVerified: true}}
	s := NewSettingsService(r, &config.Config{BootstrapToken: "installation-token-with-32-random-bytes"})
	s.Exchange = func(*config.Config) PagingExchanger { return x }
	return s, r, x
}

func TestSettingsServiceRedactionAndDraftDefaults(t *testing.T) {
	s, r, _ := settingsServiceFixture()
	ctx := context.Background()
	r.doc.Values["SESSION_SECRET"] = "legacy-secret"
	r.doc.Values["WEBHOOK_SECRET"] = "webhook-secret"
	redacted := RedactedSettings(r.doc)
	require.NotContains(t, redacted["values"], "SESSION_SECRET")
	require.Equal(t, "", redacted["values"].(map[string]string)["WEBHOOK_SECRET"])
	require.True(t, redacted["secret_present"].(map[string]bool)["WEBHOOK_SECRET"])
	tested := int64(1)
	r.draft.TestedRevision = &tested
	require.Equal(t, true, RedactedDraft(r.draft)["tested"])
	d, err := s.Draft(ctx, "google")
	require.NoError(t, err)
	require.Equal(t, int64(1), d.Revision)
	_, err = s.Draft(ctx, "unknown")
	require.Error(t, err)
	r.draftErr = pgx.ErrNoRows
	delete(r.doc.Values, "GOOGLE_OIDC_REDIRECT_URL")
	d, err = s.Draft(ctx, "google")
	require.NoError(t, err)
	require.Equal(t, "http://localhost:3000/auth/google/callback", d.Values["GOOGLE_OIDC_REDIRECT_URL"])
	r.getErr = errors.New("offline")
	_, err = s.Draft(ctx, "google")
	require.Error(t, err)
}

func TestSettingsServiceBootstrapToken(t *testing.T) {
	s, r, _ := settingsServiceFixture()
	ctx := context.Background()
	available, err := s.BootstrapAvailable(ctx)
	require.NoError(t, err)
	require.True(t, available)
	_, err = s.OpenBootstrap(ctx, "wrong", "https://aegis.test")
	require.Error(t, err)
	token, err := s.OpenBootstrap(ctx, s.Boot.BootstrapToken, "https://aegis.test")
	require.NoError(t, err)
	require.NotEmpty(t, token)
	r.available = false
	_, err = s.OpenBootstrap(ctx, s.Boot.BootstrapToken, "https://aegis.test")
	require.Error(t, err)
	r.bootstrapErr = errors.New("offline")
	_, err = s.OpenBootstrap(ctx, s.Boot.BootstrapToken, "https://aegis.test")
	require.Error(t, err)
	s.Boot.BootstrapToken = ""
	available, err = s.BootstrapAvailable(ctx)
	require.NoError(t, err)
	require.False(t, available)
}

func TestSettingsServiceVerifiedFlowsAndReplay(t *testing.T) {
	for _, kind := range []string{"login", "provider_test", "bootstrap"} {
		t.Run(kind, func(t *testing.T) {
			s, r, x := settingsServiceFixture()
			ctx := context.Background()
			session := "admin-browser"
			_, state, err := s.Start(ctx, "google", kind, session, "ADMIN@company.com")
			require.NoError(t, err)
			require.NotEmpty(t, x.nonce)
			require.WithinDuration(t, time.Now().Add(5*time.Minute), r.attempt.ExpiresAt, time.Second)
			if kind == "login" {
				session = state
			}
			a, _, token, err := s.Complete(ctx, "google", state, session, "code", "")
			require.NoError(t, err)
			require.Equal(t, kind, a.Kind)
			require.True(t, r.deleted)
			require.Equal(t, kind == "provider_test", r.finished)
			require.Equal(t, kind == "bootstrap", r.installed)
			if kind == "bootstrap" {
				require.NotEmpty(t, token)
			}
			_, _, _, err = s.Complete(ctx, "google", state, session, "code", "")
			require.Error(t, err)
		})
	}
}

func TestSettingsServiceStartFailures(t *testing.T) {
	for _, scenario := range []string{"read", "invalid_config", "no_draft", "email", "draft_config", "expired_bootstrap", "bootstrap_error", "disabled", "upstream", "write"} {
		t.Run(scenario, func(t *testing.T) {
			s, r, x := settingsServiceFixture()
			kind := "provider_test"
			switch scenario {
			case "read":
				r.getErr = errors.New("offline")
			case "invalid_config":
				r.doc.Values["SESSION_TTL"] = "bad"
			case "no_draft":
				r.draftErr = pgx.ErrNoRows
			case "email":
				r.draft.ExpectedEmail = "other@company.com"
			case "draft_config":
				r.draft.Values["GOOGLE_OIDC_REDIRECT_URL"] = "invalid"
			case "expired_bootstrap":
				kind = "bootstrap"
				r.valid = false
			case "bootstrap_error":
				kind = "bootstrap"
				r.bootstrapErr = errors.New("offline")
			case "disabled":
				kind = "login"
				r.doc.Enabled["google"] = false
			case "upstream":
				x.urlErr = errors.New("private upstream secret")
			case "write":
				r.createErr = errors.New("offline")
			}
			_, _, err := s.Start(context.Background(), "google", kind, "browser", "admin@company.com")
			require.Error(t, err)
			require.NotContains(t, err.Error(), "private upstream secret")
		})
	}
}

func TestSettingsServiceCompletionFailures(t *testing.T) {
	for _, scenario := range []string{"empty", "session", "provider", "expiry", "cancel", "no_code", "read", "revision", "invalid_config", "draft_read", "draft_revision", "draft_config", "exchange", "unverified", "empty_email", "empty_subject", "unknown_subject", "wrong_email", "finish"} {
		t.Run(scenario, func(t *testing.T) {
			s, r, x := settingsServiceFixture()
			ctx := context.Background()
			session := "browser"
			_, state, err := s.Start(ctx, "google", "provider_test", session, "admin@company.com")
			require.NoError(t, err)
			code, provider, providerError := "code", "google", ""
			switch scenario {
			case "empty":
				state = ""
			case "session":
				session = ""
			case "provider":
				provider = "slack"
			case "expiry":
				r.attempt.ExpiresAt = time.Now().Add(-time.Second)
			case "cancel":
				providerError = "access_denied"
			case "no_code":
				code = ""
			case "read":
				r.getErr = errors.New("offline")
			case "revision":
				r.doc.Revision++
			case "invalid_config":
				r.doc.Values["SESSION_TTL"] = "bad"
			case "draft_read":
				r.draftErr = pgx.ErrNoRows
			case "draft_revision":
				r.draft.Revision++
			case "draft_config":
				r.draft.Values["GOOGLE_OIDC_REDIRECT_URL"] = "bad"
			case "exchange":
				x.exchangeErr = errors.New("private upstream secret")
			case "unverified":
				x.info.EmailVerified = false
			case "empty_email":
				x.info.Email = ""
			case "empty_subject":
				x.info.Subject = ""
			case "unknown_subject":
				x.info.Subject = "unknown"
			case "wrong_email":
				x.info.Email = "other@company.com"
			case "finish":
				r.finishErr = errors.New("offline")
			}
			_, _, _, err = s.Complete(ctx, provider, state, session, code, providerError)
			require.Error(t, err)
			require.False(t, r.finished)
			require.False(t, r.installed)
			require.NotContains(t, err.Error(), "private upstream secret")
		})
	}
}

func TestSettingsServiceTestDoesNotCreateIdentity(t *testing.T) {
	s, r, _ := settingsServiceFixture()
	_, state, err := s.Start(context.Background(), "google", "provider_test", "browser", "admin@company.com")
	require.NoError(t, err)
	require.Equal(t, sessiontoken.Hash("browser"), r.attempt.SessionHash)
	_, _, token, err := s.Complete(context.Background(), "google", state, "browser", "code", "")
	require.NoError(t, err)
	require.Empty(t, token)
	require.False(t, r.installed)
}
