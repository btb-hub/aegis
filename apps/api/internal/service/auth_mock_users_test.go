package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/aegis/aegis/pkg/db"
	"github.com/google/uuid"
)

type identityMockUsers struct {
	users      map[uuid.UUID]db.User
	identities map[string]db.UserIdentity
	byEmail    map[string]uuid.UUID
	resolveErr error
}

func newIdentityMockUsers() *identityMockUsers {
	return &identityMockUsers{
		users:      map[uuid.UUID]db.User{},
		identities: map[string]db.UserIdentity{},
		byEmail:    map[string]uuid.UUID{},
	}
}

func identityKey(provider, providerSub string) string {
	return provider + ":" + providerSub
}

func normalizeEmailMock(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func (m *identityMockUsers) ResolveOIDCLogin(ctx context.Context, input db.OIDCLoginInput) (db.OIDCLoginResult, error) {
	if m.resolveErr != nil {
		return db.OIDCLoginResult{}, m.resolveErr
	}
	key := identityKey(input.Provider, input.ProviderSub)
	if existing, ok := m.identities[key]; ok {
		user := m.users[existing.UserID]
		user = backfillMockUser(user, input)
		m.users[user.ID] = user
		return db.OIDCLoginResult{
			User:              user,
			Identities:        m.listIdentities(user.ID),
			NewIdentityLinked: false,
		}, nil
	}

	email := normalizeEmailMock(input.Email)
	if email != "" {
		if userID, ok := m.byEmail[email]; ok {
			user := m.users[userID]
			m.identities[key] = db.UserIdentity{
				ID:          uuid.New(),
				UserID:      user.ID,
				Provider:    input.Provider,
				ProviderSub: input.ProviderSub,
				LinkedAt:    time.Now(),
			}
			user = backfillMockUser(user, input)
			m.users[user.ID] = user
			return db.OIDCLoginResult{
				User:              user,
				Identities:        m.listIdentities(user.ID),
				NewIdentityLinked: true,
			}, nil
		}
	}

	user := db.User{
		ID:          uuid.New(),
		Provider:    input.Provider,
		ProviderSub: input.ProviderSub,
		Email:       strings.TrimSpace(input.Email),
		DisplayName: strings.TrimSpace(input.DisplayName),
		Role:        "member",
		Locale:      "en",
	}
	if v := strings.TrimSpace(input.AvatarURL); v != "" {
		user.AvatarURL = &v
	}
	if v := strings.TrimSpace(input.SlackUserID); v != "" {
		user.SlackUserID = &v
	}
	m.users[user.ID] = user
	if email != "" {
		m.byEmail[email] = user.ID
	}
	m.identities[key] = db.UserIdentity{
		ID:          uuid.New(),
		UserID:      user.ID,
		Provider:    input.Provider,
		ProviderSub: input.ProviderSub,
		LinkedAt:    time.Now(),
	}
	return db.OIDCLoginResult{
		User:              user,
		Identities:        m.listIdentities(user.ID),
		NewIdentityLinked: true,
	}, nil
}

func backfillMockUser(user db.User, input db.OIDCLoginInput) db.User {
	if user.DisplayName == "" && strings.TrimSpace(input.DisplayName) != "" {
		user.DisplayName = strings.TrimSpace(input.DisplayName)
	}
	if (user.AvatarURL == nil || *user.AvatarURL == "") && strings.TrimSpace(input.AvatarURL) != "" {
		v := strings.TrimSpace(input.AvatarURL)
		user.AvatarURL = &v
	}
	if (user.SlackUserID == nil || *user.SlackUserID == "") && strings.TrimSpace(input.SlackUserID) != "" {
		v := strings.TrimSpace(input.SlackUserID)
		user.SlackUserID = &v
	}
	return user
}

func (m *identityMockUsers) listIdentities(userID uuid.UUID) []db.UserIdentity {
	var items []db.UserIdentity
	for _, identity := range m.identities {
		if identity.UserID == userID {
			items = append(items, identity)
		}
	}
	return items
}

func (m *identityMockUsers) UpsertDevUser(ctx context.Context, email, displayName, role, locale string) (db.User, error) {
	key := identityKey("dev", "dev-local")
	if identity, ok := m.identities[key]; ok {
		user := m.users[identity.UserID]
		user.Email = email
		user.DisplayName = displayName
		user.Role = role
		user.Locale = locale
		m.users[user.ID] = user
		return user, nil
	}
	user := db.User{
		ID:          uuid.New(),
		Provider:    "dev",
		ProviderSub: "dev-local",
		Email:       email,
		DisplayName: displayName,
		Role:        role,
		Locale:      locale,
	}
	m.users[user.ID] = user
	m.identities[key] = db.UserIdentity{
		ID: uuid.New(), UserID: user.ID, Provider: "dev", ProviderSub: "dev-local", LinkedAt: time.Now(),
	}
	return user, nil
}

func (m *identityMockUsers) GetUserByID(ctx context.Context, id uuid.UUID) (db.User, error) {
	user, ok := m.users[id]
	if !ok {
		return db.User{}, errors.New("not found")
	}
	return user, nil
}

func (m *identityMockUsers) ListUserIdentities(ctx context.Context, userID uuid.UUID) ([]db.UserIdentity, error) {
	return m.listIdentities(userID), nil
}

func (m *identityMockUsers) UpdateUserLocale(ctx context.Context, id uuid.UUID, locale string) (db.User, error) {
	user := m.users[id]
	user.Locale = locale
	m.users[id] = user
	return user, nil
}
