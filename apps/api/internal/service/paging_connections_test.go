package service

import (
	"context"
	"errors"
	"testing"

	"github.com/aegis/aegis/apps/api/internal/oidc"
	"github.com/aegis/aegis/pkg/apperrors"
	"github.com/aegis/aegis/pkg/config"
	"github.com/aegis/aegis/pkg/db"
	"github.com/aegis/aegis/pkg/sessiontoken"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type pagingTestRepo struct {
	items                          map[string]db.Integration
	attempt                        db.PagingAuthorization
	claimed                        bool
	identity                       string
	disconnected                   string
	getErr, createErr, completeErr error
}

func (r *pagingTestRepo) GetIntegrationByKind(_ context.Context, kind string) (db.Integration, error) {
	if r.getErr != nil {
		return db.Integration{}, r.getErr
	}
	item, ok := r.items[kind]
	if !ok {
		return item, pgx.ErrNoRows
	}
	return item, nil
}
func (r *pagingTestRepo) CreatePagingAuthorization(_ context.Context, a db.PagingAuthorization) error {
	r.attempt = a
	r.claimed = false
	return r.createErr
}
func (r *pagingTestRepo) ClaimPagingAuthorization(_ context.Context, stateHash, sessionHash, provider string) (db.PagingAuthorization, error) {
	if r.claimed || stateHash != r.attempt.StateHash || sessionHash != r.attempt.SessionHash || provider != r.attempt.Provider {
		return db.PagingAuthorization{}, errors.New("invalid")
	}
	r.claimed = true
	return r.attempt, nil
}
func (r *pagingTestRepo) DeletePagingAuthorization(_ context.Context, _ string) error { return nil }
func (r *pagingTestRepo) CompletePagingAuthorization(_ context.Context, _ db.PagingAuthorization, identity string) error {
	if r.completeErr == nil {
		r.identity = identity
	}
	return r.completeErr
}
func (r *pagingTestRepo) DisconnectPagingIdentity(_ context.Context, _ uuid.UUID, provider string) error {
	r.disconnected = provider
	return nil
}

type pagingTestExchange struct {
	state, nonce, team  string
	info                oidc.PagingIdentity
	urlErr, exchangeErr error
	exchanged           bool
}

func (x *pagingTestExchange) AuthorizationURL(_ context.Context, _, state, nonce, team string) (string, error) {
	x.state, x.nonce, x.team = state, nonce, team
	return "https://provider.test/authorize", x.urlErr
}
func (x *pagingTestExchange) ExchangePaging(_ context.Context, _, _, nonce string) (oidc.PagingIdentity, error) {
	x.exchanged = true
	if nonce != x.nonce {
		return oidc.PagingIdentity{}, errors.New("nonce mismatch")
	}
	return x.info, x.exchangeErr
}

type pagingTestDirectory struct {
	team, huid, email    string
	slackErr, expressErr error
}

func (d *pagingTestDirectory) SlackWorkspace(_ context.Context, _ db.Integration) (string, error) {
	return d.team, d.slackErr
}
func (d *pagingTestDirectory) ExpressHUID(_ context.Context, _ db.Integration, email string) (string, error) {
	d.email = email
	return d.huid, d.expressErr
}

func pagingFixture() (*PagingService, *pagingTestRepo, *pagingTestExchange, *pagingTestDirectory) {
	cfg := &config.Config{PublicURL: "https://aegis.test", OIDC: map[string]config.OIDCProvider{
		"slack":   {ClientID: "client", ClientSecret: "secret", RedirectURL: "https://aegis.test/auth/slack/callback", Issuer: "https://slack.com"},
		"express": {ClientID: "client", ClientSecret: "secret", RedirectURL: "https://aegis.test/auth/express/callback", Issuer: "https://sso.test"},
	}}
	repo := &pagingTestRepo{items: map[string]db.Integration{
		"slack":   {Kind: "slack", Enabled: true, Config: []byte(`{"bot_token":"bot","signing_secret":"secret"}`)},
		"express": {Kind: "express", Enabled: true, Config: []byte(`{"bot_id":"bot","host":"https://cts.test","secret_key":"secret"}`)},
	}}
	exchange := &pagingTestExchange{info: oidc.PagingIdentity{Subject: "U123", SlackUserID: "U123", SlackTeamID: "T123", Email: "authorized@example.com", EmailVerified: true}}
	directory := &pagingTestDirectory{team: "T123", huid: uuid.NewString()}
	service := NewPagingService(cfg, repo, exchange)
	service.directory = directory
	return service, repo, exchange, directory
}

func TestPagingServiceConnectAndDisconnect(t *testing.T) {
	for _, provider := range []string{"slack", "express"} {
		t.Run(provider, func(t *testing.T) {
			svc, repo, exchange, directory := pagingFixture()
			user := uuid.New()
			authorization, err := svc.Authorize(context.Background(), user, "original-session", provider)
			require.NoError(t, err)
			require.Equal(t, "https://provider.test/authorize", authorization)
			require.Equal(t, user, repo.attempt.UserID)
			require.Equal(t, sessiontoken.Hash("original-session"), repo.attempt.SessionHash)
			require.Equal(t, sessiontoken.Hash(exchange.state), repo.attempt.StateHash)
			require.NotEmpty(t, exchange.nonce)
			if provider == "slack" {
				require.Equal(t, "T123", exchange.team)
			}
			require.NoError(t, svc.Complete(context.Background(), "original-session", provider, exchange.state, "code", ""))
			if provider == "slack" {
				require.Equal(t, "U123", repo.identity)
			} else {
				require.Equal(t, directory.huid, repo.identity)
				require.Equal(t, "authorized@example.com", directory.email)
			}
			require.Error(t, svc.Complete(context.Background(), "original-session", provider, exchange.state, "code", ""))
			require.NoError(t, svc.Disconnect(context.Background(), user, provider))
			require.Equal(t, provider, repo.disconnected)
		})
	}
}

func TestPagingServiceFailuresPreserveIdentity(t *testing.T) {
	for _, test := range []struct {
		name, provider, session, code, denied, expected string
		mutate                                          func(*pagingTestRepo, *pagingTestExchange, *pagingTestDirectory)
	}{
		{name: "session changed", provider: "slack", session: "changed", code: "code", expected: "invalid_authorization"},
		{name: "missing session", provider: "slack", expected: "invalid_authorization"},
		{name: "cancel", provider: "slack", session: "session", denied: "access_denied", expected: "cancelled"},
		{name: "no code", provider: "slack", session: "session", expected: "invalid_authorization"},
		{name: "bad token", provider: "slack", session: "session", code: "code", expected: "invalid_authorization", mutate: func(_ *pagingTestRepo, x *pagingTestExchange, _ *pagingTestDirectory) {
			x.exchangeErr = errors.New("bad token")
		}},
		{name: "wrong workspace", provider: "slack", session: "session", code: "code", expected: "workspace_mismatch", mutate: func(_ *pagingTestRepo, x *pagingTestExchange, _ *pagingTestDirectory) { x.info.SlackTeamID = "Twrong" }},
		{name: "missing user", provider: "slack", session: "session", code: "code", expected: "invalid_authorization", mutate: func(_ *pagingTestRepo, x *pagingTestExchange, _ *pagingTestDirectory) { x.info.SlackUserID = "" }},
		{name: "bot unavailable", provider: "slack", session: "session", code: "code", expected: "provider_unavailable", mutate: func(_ *pagingTestRepo, _ *pagingTestExchange, d *pagingTestDirectory) {
			d.slackErr = errors.New("offline")
		}},
		{name: "disabled during exchange", provider: "slack", session: "session", code: "code", expected: "bot_disabled", mutate: func(r *pagingTestRepo, _ *pagingTestExchange, _ *pagingTestDirectory) {
			item := r.items["slack"]
			item.Enabled = false
			r.items["slack"] = item
		}},
		{name: "unverified email", provider: "express", session: "session", code: "code", expected: "email_unverified", mutate: func(_ *pagingTestRepo, x *pagingTestExchange, _ *pagingTestDirectory) { x.info.EmailVerified = false }},
		{name: "missing email", provider: "express", session: "session", code: "code", expected: "email_unverified", mutate: func(_ *pagingTestRepo, x *pagingTestExchange, _ *pagingTestDirectory) { x.info.Email = "" }},
		{name: "directory mismatch", provider: "express", session: "session", code: "code", expected: "email_match_failed", mutate: func(_ *pagingTestRepo, _ *pagingTestExchange, d *pagingTestDirectory) {
			d.expressErr = errors.New("no match")
		}},
		{name: "bad huid", provider: "express", session: "session", code: "code", expected: "email_match_failed", mutate: func(_ *pagingTestRepo, _ *pagingTestExchange, d *pagingTestDirectory) { d.huid = "bad" }},
		{name: "claimed identity", provider: "express", session: "session", code: "code", expected: "identity_in_use", mutate: func(r *pagingTestRepo, _ *pagingTestExchange, _ *pagingTestDirectory) {
			r.completeErr = apperrors.Conflict("claimed")
		}},
		{name: "disconnected during exchange", provider: "express", session: "session", code: "code", expected: "invalid_authorization", mutate: func(r *pagingTestRepo, _ *pagingTestExchange, _ *pagingTestDirectory) {
			r.completeErr = apperrors.Validation("expired", nil)
		}},
		{name: "database error", provider: "express", session: "session", code: "code", expected: "database failed", mutate: func(r *pagingTestRepo, _ *pagingTestExchange, _ *pagingTestDirectory) {
			r.completeErr = errors.New("database failed")
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			svc, repo, exchange, directory := pagingFixture()
			_, err := svc.Authorize(context.Background(), uuid.New(), "session", test.provider)
			require.NoError(t, err)
			repo.identity = "original"
			if test.mutate != nil {
				test.mutate(repo, exchange, directory)
			}
			err = svc.Complete(context.Background(), test.session, test.provider, exchange.state, test.code, test.denied)
			require.ErrorContains(t, err, test.expected)
			require.Equal(t, "original", repo.identity)
			if test.denied != "" {
				require.False(t, exchange.exchanged)
			}
		})
	}
}

func TestPagingAvailabilityAndStartErrors(t *testing.T) {
	ctx := context.Background()
	for _, test := range []struct {
		name     string
		mutate   func(*PagingService, *pagingTestRepo, *pagingTestExchange, *pagingTestDirectory)
		expected string
	}{
		{name: "configured"},
		{name: "no OIDC", expected: "authentication_required", mutate: func(s *PagingService, _ *pagingTestRepo, _ *pagingTestExchange, _ *pagingTestDirectory) {
			delete(s.cfg.OIDC, "slack")
		}},
		{name: "no bot", expected: "bot_required", mutate: func(_ *PagingService, r *pagingTestRepo, _ *pagingTestExchange, _ *pagingTestDirectory) {
			delete(r.items, "slack")
		}},
		{name: "disabled", expected: "bot_disabled", mutate: func(_ *PagingService, r *pagingTestRepo, _ *pagingTestExchange, _ *pagingTestDirectory) {
			i := r.items["slack"]
			i.Enabled = false
			r.items["slack"] = i
		}},
		{name: "bad bot config", expected: "bot_required", mutate: func(_ *PagingService, r *pagingTestRepo, _ *pagingTestExchange, _ *pagingTestDirectory) {
			i := r.items["slack"]
			i.Config = []byte(`{}`)
			r.items["slack"] = i
		}},
		{name: "database", expected: "db failed", mutate: func(_ *PagingService, r *pagingTestRepo, _ *pagingTestExchange, _ *pagingTestDirectory) {
			r.getErr = errors.New("db failed")
		}},
		{name: "offline bot", expected: "provider_unavailable", mutate: func(_ *PagingService, _ *pagingTestRepo, _ *pagingTestExchange, d *pagingTestDirectory) {
			d.slackErr = errors.New("offline")
		}},
		{name: "offline OIDC", expected: "provider_unavailable", mutate: func(_ *PagingService, _ *pagingTestRepo, x *pagingTestExchange, _ *pagingTestDirectory) {
			x.urlErr = errors.New("offline")
		}},
		{name: "save failed", expected: "db failed", mutate: func(_ *PagingService, r *pagingTestRepo, _ *pagingTestExchange, _ *pagingTestDirectory) {
			r.createErr = errors.New("db failed")
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			svc, repo, exchange, directory := pagingFixture()
			if test.mutate != nil {
				test.mutate(svc, repo, exchange, directory)
			}
			connections, err := svc.Connections(ctx, db.User{})
			if repo.getErr != nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Len(t, connections, 2)
				_, setupErr := svc.integration(ctx, "slack")
				require.Equal(t, setupErr == nil, connections[0].Available)
			}
			_, err = svc.Authorize(ctx, uuid.New(), "session", "slack")
			if test.expected == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, test.expected)
			}
		})
	}
	svc, _, _, _ := pagingFixture()
	_, err := svc.Authorize(ctx, uuid.New(), "session", "google")
	require.ErrorContains(t, err, "unknown_provider")
	require.ErrorContains(t, svc.Disconnect(ctx, uuid.New(), "google"), "unknown_provider")
	require.ErrorContains(t, svc.Complete(ctx, "session", "slack", "non-paging-state", "code", ""), "invalid_authorization")
}
