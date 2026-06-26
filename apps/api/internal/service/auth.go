package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aegis/aegis/apps/api/internal/oidc"
	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/aegis/aegis/pkg/config"
	"github.com/aegis/aegis/pkg/db"
	"github.com/aegis/aegis/pkg/locale"
	"github.com/aegis/aegis/pkg/sessiontoken"
	"github.com/google/uuid"
)

type UserRepository interface {
	UpsertUser(ctx context.Context, provider, providerSub, email, displayName, role, locale string) (db.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (db.User, error)
	UpdateUserLocale(ctx context.Context, id uuid.UUID, locale string) (db.User, error)
}

type SessionRepository interface {
	CreateSession(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (db.Session, error)
	GetSessionByTokenHash(ctx context.Context, tokenHash string) (db.Session, error)
	DeleteSession(ctx context.Context, tokenHash string) error
}

type OIDCUserInfo = oidc.UserInfo

type TokenExchanger interface {
	AuthCodeURL(provider string, state string) (string, error)
	Exchange(ctx context.Context, provider, code string) (*OIDCUserInfo, error)
}

type AuthService struct {
	cfg      *config.Config
	users    UserRepository
	sessions SessionRepository
	oidc     TokenExchanger
	now      func() time.Time
	newState func() (string, error)
}

func NewAuthService(cfg *config.Config, users UserRepository, sessions SessionRepository, oidc TokenExchanger) *AuthService {
	return &AuthService{
		cfg:      cfg,
		users:    users,
		sessions: sessions,
		oidc:     oidc,
		now:      time.Now,
		newState: randomState,
	}
}

func (s *AuthService) LoginURL(provider string) (string, string, error) {
	if _, err := s.cfg.Provider(provider); err != nil {
		return "", "", apperrors.Validation("unknown or unconfigured provider", map[string]any{"provider": provider})
	}
	state, err := s.newState()
	if err != nil {
		return "", "", fmt.Errorf("state: %w", err)
	}
	url, err := s.oidc.AuthCodeURL(provider, state)
	if err != nil {
		return "", "", err
	}
	return url, state, nil
}

func (s *AuthService) CompleteLogin(ctx context.Context, provider, code string) (token string, user db.User, err error) {
	info, err := s.oidc.Exchange(ctx, provider, code)
	if err != nil {
		return "", db.User{}, apperrors.Validation("oidc exchange failed", map[string]any{"error": err.Error()})
	}

	user, err = s.users.UpsertUser(ctx, provider, info.Sub, info.Email, info.DisplayName, "member", "en")
	if err != nil {
		return "", db.User{}, err
	}

	rawToken, hash, err := sessiontoken.New()
	if err != nil {
		return "", db.User{}, err
	}
	expires := s.now().Add(s.cfg.SessionTTL)
	if _, err := s.sessions.CreateSession(ctx, user.ID, hash, expires); err != nil {
		return "", db.User{}, err
	}
	return rawToken, user, nil
}

func (s *AuthService) Logout(ctx context.Context, token string) error {
	if token == "" {
		return apperrors.Unauthorized("missing session")
	}
	return s.sessions.DeleteSession(ctx, sessiontoken.Hash(token))
}

func (s *AuthService) CurrentUser(ctx context.Context, token string) (db.User, error) {
	if token == "" {
		return db.User{}, apperrors.Unauthorized("missing session")
	}
	session, err := s.sessions.GetSessionByTokenHash(ctx, sessiontoken.Hash(token))
	if err != nil {
		return db.User{}, apperrors.Unauthorized("invalid session")
	}
	return s.users.GetUserByID(ctx, session.UserID)
}

func (s *AuthService) UpdateLocale(ctx context.Context, token, newLocale string) (db.User, error) {
	if err := locale.Validate(newLocale); err != nil {
		return db.User{}, apperrors.InvalidLocale()
	}
	user, err := s.CurrentUser(ctx, token)
	if err != nil {
		return db.User{}, err
	}
	return s.users.UpdateUserLocale(ctx, user.ID, newLocale)
}

func randomState() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func NewOAuthTokenExchanger(cfg *config.Config) TokenExchanger {
	return oidc.NewClient(cfg)
}

func UserJSON(user db.User) map[string]any {
	return map[string]any{
		"id":           user.ID.String(),
		"email":        user.Email,
		"display_name": user.DisplayName,
		"role":         user.Role,
		"locale":       user.Locale,
		"provider":     user.Provider,
	}
}

func MarshalUser(user db.User) ([]byte, error) {
	return json.Marshal(UserJSON(user))
}
