package service

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/aegis/aegis/apps/api/internal/oidc"
	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/aegis/aegis/pkg/config"
	"github.com/aegis/aegis/pkg/db"
	intexpress "github.com/aegis/aegis/pkg/integrations/express"
	intslack "github.com/aegis/aegis/pkg/integrations/slack"
	"github.com/aegis/aegis/pkg/sessiontoken"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type PagingRepository interface {
	GetIntegrationByKind(ctx context.Context, kind string) (db.Integration, error)
	CreatePagingAuthorization(ctx context.Context, attempt db.PagingAuthorization) error
	ClaimPagingAuthorization(ctx context.Context, stateHash, sessionHash, provider string) (db.PagingAuthorization, error)
	DeletePagingAuthorization(ctx context.Context, stateHash string) error
	CompletePagingAuthorization(ctx context.Context, attempt db.PagingAuthorization, identity string) error
	DisconnectPagingIdentity(ctx context.Context, userID uuid.UUID, provider string) error
}

type PagingExchanger interface {
	AuthorizationURL(ctx context.Context, provider, state, nonce, team string) (string, error)
	ExchangePaging(ctx context.Context, provider, code, nonce string) (oidc.PagingIdentity, error)
}

type PagingDirectory interface {
	SlackWorkspace(ctx context.Context, integration db.Integration) (string, error)
	ExpressHUID(ctx context.Context, integration db.Integration, email string) (string, error)
}

type botPagingDirectory struct{ publicURL string }

func (d botPagingDirectory) SlackWorkspace(ctx context.Context, integration db.Integration) (string, error) {
	provider, err := intslack.NewFromJSON(integration.Config, d.publicURL)
	if err != nil {
		return "", err
	}
	return provider.WorkspaceID(ctx)
}

func (d botPagingDirectory) ExpressHUID(ctx context.Context, integration db.Integration, email string) (string, error) {
	provider, err := intexpress.NewFromJSON(integration.Config)
	if err != nil {
		return "", err
	}
	return provider.LookupPagingHUID(ctx, email)
}

type PagingService struct {
	cfg       *config.Config
	repo      PagingRepository
	exchanger PagingExchanger
	directory PagingDirectory
}

func NewPagingService(cfg *config.Config, repo PagingRepository, exchanger PagingExchanger) *PagingService {
	return &PagingService{cfg: cfg, repo: repo, exchanger: exchanger, directory: botPagingDirectory{cfg.PublicURL}}
}

type PagingConnection struct {
	Provider          string  `json:"provider"`
	Identity          *string `json:"identity"`
	Available         bool    `json:"available"`
	UnavailableReason string  `json:"unavailable_reason,omitempty"`
}

func pagingProvider(provider string) bool { return provider == "slack" || provider == "express" }

func PagingError(code string) *apperrors.Error {
	return apperrors.New("PAGING_"+strings.ToUpper(code), code, http.StatusBadRequest)
}

func (s *PagingService) integration(ctx context.Context, provider string) (db.Integration, error) {
	if !pagingProvider(provider) {
		return db.Integration{}, PagingError("unknown_provider")
	}
	if _, err := s.cfg.Provider(provider); err != nil {
		return db.Integration{}, PagingError("setup_required")
	}
	item, err := s.repo.GetIntegrationByKind(ctx, provider)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return item, PagingError("setup_required")
		}
		return item, err
	}
	if !item.Enabled || item.WorkspaceID != nil {
		return item, PagingError("setup_required")
	}
	if err := parseProviderConfig(provider, item.Config, s.cfg.PublicURL); err != nil {
		return item, PagingError("setup_required")
	}
	return item, nil
}

func (s *PagingService) Connections(ctx context.Context, user db.User) ([]PagingConnection, error) {
	connections := []PagingConnection{{Provider: "slack", Identity: user.SlackUserID}, {Provider: "express", Identity: db.ExpressHuidString(user)}}
	for i := range connections {
		_, err := s.integration(ctx, connections[i].Provider)
		if err == nil {
			connections[i].Available = true
			continue
		}
		var appErr *apperrors.Error
		if !errors.As(err, &appErr) {
			return nil, err
		}
		connections[i].UnavailableReason = appErr.Message
	}
	return connections, nil
}

func (s *PagingService) Authorize(ctx context.Context, userID uuid.UUID, session, provider string) (string, error) {
	integration, err := s.integration(ctx, provider)
	if err != nil {
		return "", err
	}
	team := ""
	if provider == "slack" {
		team, err = s.directory.SlackWorkspace(ctx, integration)
		if err != nil {
			return "", PagingError("provider_unavailable")
		}
	}
	state, err := randomState()
	if err != nil {
		return "", err
	}
	nonce, err := randomState()
	if err != nil {
		return "", err
	}
	// A distinct prefix ensures expired paging callbacks never fall through to login.
	state = "paging." + state
	authorizationURL, err := s.exchanger.AuthorizationURL(ctx, provider, state, nonce, team)
	if err != nil {
		return "", PagingError("provider_unavailable")
	}
	err = s.repo.CreatePagingAuthorization(ctx, db.PagingAuthorization{StateHash: sessiontoken.Hash(state), UserID: userID,
		SessionHash: sessiontoken.Hash(session), Provider: provider, Nonce: nonce, ExpiresAt: time.Now().Add(5 * time.Minute)})
	return authorizationURL, err
}

func (s *PagingService) Complete(ctx context.Context, session, provider, state, code, providerError string) error {
	if !pagingProvider(provider) || session == "" || !strings.HasPrefix(state, "paging.") {
		return PagingError("invalid_authorization")
	}
	attempt, err := s.repo.ClaimPagingAuthorization(ctx, sessiontoken.Hash(state), sessiontoken.Hash(session), provider)
	if err != nil {
		return PagingError("invalid_authorization")
	}
	defer s.repo.DeletePagingAuthorization(context.WithoutCancel(ctx), attempt.StateHash)
	if providerError != "" {
		return PagingError("cancelled")
	}
	if code == "" {
		return PagingError("invalid_authorization")
	}
	integration, err := s.integration(ctx, provider)
	if err != nil {
		return err
	}
	info, err := s.exchanger.ExchangePaging(ctx, provider, code, attempt.Nonce)
	if err != nil {
		return PagingError("invalid_authorization")
	}
	identity := ""
	if provider == "slack" {
		team, err := s.directory.SlackWorkspace(ctx, integration)
		if err != nil {
			return PagingError("provider_unavailable")
		}
		if info.SlackTeamID == "" || info.SlackTeamID != team {
			return PagingError("workspace_mismatch")
		}
		if info.SlackUserID == "" || info.SlackUserID != info.Subject {
			return PagingError("invalid_authorization")
		}
		identity = info.SlackUserID
	} else {
		if !info.EmailVerified || strings.TrimSpace(info.Email) == "" {
			return PagingError("email_unverified")
		}
		identity, err = s.directory.ExpressHUID(ctx, integration, info.Email)
		if err != nil {
			return PagingError("email_match_failed")
		}
		huid, err := uuid.Parse(identity)
		if err != nil || huid == uuid.Nil {
			return PagingError("email_match_failed")
		}
		identity = huid.String()
	}
	if err := s.repo.CompletePagingAuthorization(ctx, attempt, identity); err != nil {
		var appErr *apperrors.Error
		if errors.As(err, &appErr) && appErr.Code == "CONFLICT" {
			return PagingError("identity_in_use")
		}
		if errors.As(err, &appErr) && appErr.Code == "VALIDATION_ERROR" {
			return PagingError("invalid_authorization")
		}
		return err
	}
	return nil
}

func (s *PagingService) Disconnect(ctx context.Context, userID uuid.UUID, provider string) error {
	if !pagingProvider(provider) {
		return PagingError("unknown_provider")
	}
	return s.repo.DisconnectPagingIdentity(ctx, userID, provider)
}
