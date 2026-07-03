package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aegis/aegis/apps/api/internal/oidc"
	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/aegis/aegis/pkg/config"
	"github.com/aegis/aegis/pkg/db"
	"github.com/aegis/aegis/pkg/locale"
	"github.com/aegis/aegis/pkg/rbac"
	"github.com/aegis/aegis/pkg/sessiontoken"
	"github.com/google/uuid"
)

type UserRepository interface {
	ResolveOIDCLogin(ctx context.Context, input db.OIDCLoginInput) (db.OIDCLoginResult, error)
	UpsertDevUser(ctx context.Context, email, displayName, role, locale string) (db.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (db.User, error)
	ListUserIdentities(ctx context.Context, userID uuid.UUID) ([]db.UserIdentity, error)
	UpdateUserLocale(ctx context.Context, id uuid.UUID, locale string) (db.User, error)
	UpdateUserProfile(ctx context.Context, id uuid.UUID, displayName, locale string) (db.User, error)
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

const (
	devAuthDisplayName = "Local Dev User"
)

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

func (s *AuthService) DevAuthEnabled() bool {
	return s.cfg.DevAuthEnabled
}

func (s *AuthService) DevLogin(ctx context.Context, role string) (token string, user db.User, err error) {
	if !s.cfg.DevAuthEnabled {
		return "", db.User{}, apperrors.NotFound("dev auth")
	}
	if role == "" {
		role = s.cfg.DevAuthDefaultRole
	}
	parsedRole, err := rbac.Parse(role)
	if err != nil {
		return "", db.User{}, apperrors.Validation("invalid role", map[string]any{"role": role})
	}

	user, err = s.users.UpsertDevUser(
		ctx,
		s.cfg.DevAuthEmail,
		devAuthDisplayName,
		string(parsedRole),
		"en",
	)
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

func (s *AuthService) CompleteLogin(ctx context.Context, provider, code string) (token string, user db.User, err error) {
	info, err := s.oidc.Exchange(ctx, provider, code)
	if err != nil {
		return "", db.User{}, apperrors.Validation("oidc exchange failed", map[string]any{"error": err.Error()})
	}

	result, err := s.users.ResolveOIDCLogin(ctx, db.OIDCLoginInput{
		Provider:    provider,
		ProviderSub: info.Sub,
		Email:       info.Email,
		DisplayName: info.DisplayName,
		AvatarURL:   info.AvatarURL,
		SlackUserID: info.SlackUserID,
	})
	if err != nil {
		return "", db.User{}, err
	}
	user = result.User

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
	profile, err := s.CurrentUserProfile(ctx, token)
	if err != nil {
		return db.User{}, err
	}
	return profile.User, nil
}

type UserProfile struct {
	User       db.User
	Identities []db.UserIdentity
}

func (s *AuthService) CurrentUserProfile(ctx context.Context, token string) (UserProfile, error) {
	if token == "" {
		return UserProfile{}, apperrors.Unauthorized("missing session")
	}
	session, err := s.sessions.GetSessionByTokenHash(ctx, sessiontoken.Hash(token))
	if err != nil {
		return UserProfile{}, apperrors.Unauthorized("invalid session")
	}
	user, err := s.users.GetUserByID(ctx, session.UserID)
	if err != nil {
		return UserProfile{}, err
	}
	identities, err := s.users.ListUserIdentities(ctx, user.ID)
	if err != nil {
		return UserProfile{}, err
	}
	return UserProfile{User: user, Identities: identities}, nil
}

func (s *AuthService) UpdateLocale(ctx context.Context, token, newLocale string) (db.User, error) {
	return s.UpdateProfile(ctx, token, UpdateProfileInput{Locale: &newLocale})
}

type UpdateProfileInput struct {
	DisplayName *string
	Locale      *string
}

const maxDisplayNameLength = 120

func (s *AuthService) UpdateProfile(ctx context.Context, token string, input UpdateProfileInput) (db.User, error) {
	user, err := s.CurrentUser(ctx, token)
	if err != nil {
		return db.User{}, err
	}

	displayName := user.DisplayName
	if input.DisplayName != nil {
		trimmed := strings.TrimSpace(*input.DisplayName)
		if trimmed == "" {
			return db.User{}, apperrors.Validation("display_name must not be empty", nil)
		}
		if len(trimmed) > maxDisplayNameLength {
			return db.User{}, apperrors.Validation("display_name is too long", nil)
		}
		displayName = trimmed
	}

	localeValue := user.Locale
	if input.Locale != nil {
		if err := locale.Validate(*input.Locale); err != nil {
			return db.User{}, apperrors.InvalidLocale()
		}
		localeValue = *input.Locale
	}

	return s.users.UpdateUserProfile(ctx, user.ID, displayName, localeValue)
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

func UserJSON(user db.User, identities []db.UserIdentity) map[string]any {
	out := map[string]any{
		"id":           user.ID.String(),
		"email":        user.Email,
		"display_name": user.DisplayName,
		"role":         user.Role,
		"locale":       user.Locale,
		"provider":     user.Provider,
	}
	if user.AvatarURL != nil && *user.AvatarURL != "" {
		out["avatar_url"] = *user.AvatarURL
	}
	if user.SlackUserID != nil && *user.SlackUserID != "" {
		out["slack_user_id"] = *user.SlackUserID
	}
	if user.ExpressUserHuid.Valid {
		out["express_user_huid"] = uuid.UUID(user.ExpressUserHuid.Bytes).String()
	}
	if identities != nil {
		out["identities"] = IdentitiesJSON(identities)
	}
	return out
}

func IdentitiesJSON(identities []db.UserIdentity) []map[string]any {
	items := make([]map[string]any, 0, len(identities))
	for _, identity := range identities {
		items = append(items, map[string]any{
			"provider":     identity.Provider,
			"provider_sub": identity.ProviderSub,
			"linked_at":    identity.LinkedAt.UTC().Format(time.RFC3339),
		})
	}
	return items
}

func MarshalUser(user db.User, identities []db.UserIdentity) ([]byte, error) {
	return json.Marshal(UserJSON(user, identities))
}
