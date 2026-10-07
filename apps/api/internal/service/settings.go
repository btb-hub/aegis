package service

import (
	"context"
	"crypto/subtle"
	"errors"
	"strings"
	"time"

	"github.com/aegis/aegis/apps/api/internal/oidc"
	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/aegis/aegis/pkg/config"
	"github.com/aegis/aegis/pkg/db"
	"github.com/aegis/aegis/pkg/sessiontoken"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type SettingsRepository interface {
	GetSettings(context.Context) (config.Settings, error)
	PatchSettings(context.Context, uuid.UUID, int64, string, map[string]string, bool) (config.Settings, error)
	GetProviderDraft(context.Context, string) (db.ProviderDraft, error)
	SaveProviderDraft(context.Context, *uuid.UUID, string, int64, map[string]string, string) (db.ProviderDraft, error)
	ActivateProvider(context.Context, uuid.UUID, string, string, int64, int64, bool) (config.Settings, error)
	CreateSettingsAuthorization(context.Context, db.SettingsAuthorization) error
	ClaimSettingsAuthorization(context.Context, string, string, string) (db.SettingsAuthorization, error)
	DeleteSettingsAuthorization(context.Context, string) error
	ValidateLoginAuthorization(context.Context, db.SettingsAuthorization) error
	FinishProviderTest(context.Context, db.SettingsAuthorization) error
	BootstrapAvailable(context.Context) (bool, error)
	CreateBootstrapSession(context.Context, string, string) error
	ValidBootstrapSession(context.Context, string) (bool, error)
	CompleteBootstrap(context.Context, db.SettingsAuthorization, db.OIDCLoginInput, string) (db.User, error)
}

type SettingsService struct {
	Repo     SettingsRepository
	Boot     *config.Config
	Exchange func(*config.Config) PagingExchanger
}

func NewSettingsService(repo SettingsRepository, boot *config.Config) *SettingsService {
	return &SettingsService{Repo: repo, Boot: boot, Exchange: func(c *config.Config) PagingExchanger { return oidc.NewPagingClient(c) }}
}

func RedactedSettings(doc config.Settings) map[string]any {
	values := map[string]string{}
	secrets := map[string]bool{}
	for key, value := range doc.Values {
		if key == "SESSION_SECRET" {
			continue
		}
		if config.SecretKey(key) {
			values[key] = ""
			secrets[key] = value != ""
		} else {
			values[key] = value
		}
	}
	return map[string]any{"values": values, "secret_present": secrets, "enabled": doc.Enabled, "revision": doc.Revision, "bootstrap_closed": doc.BootstrapClosed}
}

func RedactedDraft(d db.ProviderDraft) map[string]any {
	values := map[string]string{}
	secrets := map[string]bool{}
	for key, value := range d.Values {
		if config.SecretKey(key) {
			values[key] = ""
			secrets[key] = value != ""
		} else {
			values[key] = value
		}
	}
	return map[string]any{"provider": d.Provider, "values": values, "secret_present": secrets, "revision": d.Revision, "tested": d.TestedRevision != nil && *d.TestedRevision == d.Revision, "expected_email": d.ExpectedEmail}
}

func (s *SettingsService) Draft(ctx context.Context, provider string) (db.ProviderDraft, error) {
	if provider != "google" && provider != "slack" && provider != "express" {
		return db.ProviderDraft{}, apperrors.Validation("unknown provider", nil)
	}
	d, err := s.Repo.GetProviderDraft(ctx, provider)
	if !errors.Is(err, pgx.ErrNoRows) {
		return d, err
	}
	doc, err := s.Repo.GetSettings(ctx)
	if err != nil {
		return d, err
	}
	d = db.ProviderDraft{Provider: provider, Values: map[string]string{}}
	for _, key := range config.ProviderKeys(provider) {
		d.Values[key] = doc.Values[key]
	}
	key := strings.ToUpper(provider) + "_OIDC_REDIRECT_URL"
	if d.Values[key] == "" {
		d.Values[key] = strings.TrimRight(doc.Values["PUBLIC_URL"], "/") + "/auth/" + provider + "/callback"
	}
	return d, nil
}

func (s *SettingsService) BootstrapAvailable(ctx context.Context) (bool, error) {
	if len(s.Boot.BootstrapToken) < 32 {
		return false, nil
	}
	return s.Repo.BootstrapAvailable(ctx)
}

func (s *SettingsService) OpenBootstrap(ctx context.Context, token, publicURL string) (string, error) {
	if len(s.Boot.BootstrapToken) < 32 || subtle.ConstantTimeCompare([]byte(sessiontoken.Hash(token)), []byte(sessiontoken.Hash(s.Boot.BootstrapToken))) != 1 {
		return "", apperrors.Unauthorized("invalid installation token")
	}
	available, err := s.Repo.BootstrapAvailable(ctx)
	if err != nil {
		return "", err
	}
	if !available {
		return "", apperrors.NotFound("bootstrap")
	}
	raw, hash, err := sessiontoken.New()
	if err != nil {
		return "", err
	}
	return raw, s.Repo.CreateBootstrapSession(ctx, hash, publicURL)
}

func (s *SettingsService) Start(ctx context.Context, provider, kind, session, email string) (string, string, error) {
	doc, err := s.Repo.GetSettings(ctx)
	if err != nil {
		return "", "", err
	}
	runtime, err := doc.Config(s.Boot)
	if err != nil {
		return "", "", err
	}
	a := db.SettingsAuthorization{Provider: provider, Kind: kind, SettingsRevision: doc.Revision, ExpectedEmail: strings.ToLower(strings.TrimSpace(email)), ExpiresAt: time.Now().Add(5 * time.Minute)}
	if kind != "login" {
		draft, err := s.Repo.GetProviderDraft(ctx, provider)
		if err != nil {
			return "", "", apperrors.Validation("save the provider draft before testing", nil)
		}
		if draft.ExpectedEmail != a.ExpectedEmail {
			return "", "", apperrors.Validation("administrator email does not match the draft", nil)
		}
		runtime, err = db.DraftConfig(doc, draft, s.Boot)
		if err != nil {
			return "", "", err
		}
		a.DraftRevision = draft.Revision
		if kind == "bootstrap" {
			valid, e := s.Repo.ValidBootstrapSession(ctx, sessiontoken.Hash(session))
			if e != nil {
				return "", "", e
			}
			if !valid {
				return "", "", apperrors.Unauthorized("installation session expired")
			}
		}
	}
	if _, err = runtime.Provider(provider); err != nil {
		return "", "", apperrors.Validation("sign-in provider is unavailable", nil)
	}
	state, err := randomState()
	if err != nil {
		return "", "", err
	}
	state = kind + "." + state
	nonce, err := randomState()
	if err != nil {
		return "", "", err
	}
	a.StateHash = sessiontoken.Hash(state)
	a.Nonce = nonce
	if kind == "login" {
		a.SessionHash = sessiontoken.Hash(state)
	} else {
		a.SessionHash = sessiontoken.Hash(session)
	}
	url, err := s.Exchange(runtime).AuthorizationURL(config.WithRuntime(ctx, runtime), provider, state, nonce, "")
	if err != nil {
		return "", "", apperrors.Validation("provider authorization is unavailable", nil)
	}
	if err = s.Repo.CreateSettingsAuthorization(ctx, a); err != nil {
		return "", "", err
	}
	return url, state, nil
}

// Complete verifies claims before any account, draft or installation mutation.
func (s *SettingsService) Complete(ctx context.Context, provider, state, session, code, providerError string) (db.SettingsAuthorization, oidc.PagingIdentity, string, error) {
	var info oidc.PagingIdentity
	var a db.SettingsAuthorization
	if state == "" || session == "" {
		return a, info, "", ErrInvalidOAuthState()
	}
	a, err := s.Repo.ClaimSettingsAuthorization(ctx, sessiontoken.Hash(state), sessiontoken.Hash(session), provider)
	if err != nil {
		return a, info, "", err
	}
	defer s.Repo.DeleteSettingsAuthorization(context.WithoutCancel(ctx), a.StateHash)
	if providerError != "" || code == "" {
		return a, info, "", apperrors.Validation("authorization cancelled or expired", nil)
	}
	doc, err := s.Repo.GetSettings(ctx)
	if err != nil {
		return a, info, "", err
	}
	if doc.Revision != a.SettingsRevision {
		return a, info, "", apperrors.Conflict("settings changed during authorization")
	}
	runtime, err := doc.Config(s.Boot)
	if err != nil {
		return a, info, "", err
	}
	if a.Kind != "login" {
		draft, e := s.Repo.GetProviderDraft(ctx, provider)
		if e != nil {
			return a, info, "", e
		}
		if draft.Revision != a.DraftRevision || draft.ExpectedEmail != a.ExpectedEmail {
			return a, info, "", apperrors.Conflict("provider draft changed during authorization")
		}
		runtime, err = db.DraftConfig(doc, draft, s.Boot)
		if err != nil {
			return a, info, "", err
		}
	}
	info, err = s.Exchange(runtime).ExchangePaging(config.WithRuntime(ctx, runtime), provider, code, a.Nonce)
	if err != nil {
		return a, info, "", apperrors.Validation("provider identity could not be verified", nil)
	}
	if !info.EmailVerified || strings.TrimSpace(info.Email) == "" || info.Subject == "" || info.Subject == "unknown" {
		return a, info, "", apperrors.Validation("a verified email and identity are required", nil)
	}
	if a.Kind == "login" {
		return a, info, "", s.Repo.ValidateLoginAuthorization(ctx, a)
	}
	if strings.ToLower(strings.TrimSpace(info.Email)) != a.ExpectedEmail {
		return a, info, "", apperrors.Validation("sign in with the administrator email used for this test", nil)
	}
	if a.Kind == "provider_test" {
		return a, info, "", s.Repo.FinishProviderTest(ctx, a)
	}
	raw, hash, err := sessiontoken.New()
	if err != nil {
		return a, info, "", err
	}
	_, err = s.Repo.CompleteBootstrap(ctx, a, db.OIDCLoginInput{Provider: provider, ProviderSub: info.Subject, Email: info.Email, DisplayName: info.DisplayName, AvatarURL: info.AvatarURL}, hash)
	return a, info, raw, err
}
