package oidc

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/aegis/aegis/pkg/config"
	coreoidc "github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

type PagingIdentity struct {
	Subject       string
	Email         string
	EmailVerified bool
	SlackUserID   string
	SlackTeamID   string
}

// PagingClient deliberately does not use the legacy login payload parser.
type PagingClient struct {
	cfg       *config.Config
	client    *http.Client
	mu        sync.Mutex
	providers map[string]*coreoidc.Provider
}

func NewPagingClient(cfg *config.Config) *PagingClient {
	return &PagingClient{cfg: cfg, client: &http.Client{Timeout: 15 * time.Second}, providers: make(map[string]*coreoidc.Provider)}
}

func (c *PagingClient) provider(ctx context.Context, name string) (*coreoidc.Provider, *oauth2.Config, error) {
	p, err := c.cfg.Provider(name)
	if err != nil {
		return nil, nil, err
	}
	c.mu.Lock()
	discovered := c.providers[name]
	c.mu.Unlock()
	if discovered == nil {
		discovered, err = coreoidc.NewProvider(coreoidc.ClientContext(ctx, c.client), p.Issuer)
		if err != nil {
			return nil, nil, err
		}
		c.mu.Lock()
		c.providers[name] = discovered
		c.mu.Unlock()
	}
	return discovered, &oauth2.Config{ClientID: p.ClientID, ClientSecret: p.ClientSecret,
		RedirectURL: p.RedirectURL, Endpoint: discovered.Endpoint(), Scopes: []string{"openid", "email", "profile"}}, nil
}

func (c *PagingClient) AuthorizationURL(ctx context.Context, provider, state, nonce, team string) (string, error) {
	_, cfg, err := c.provider(ctx, provider)
	if err != nil {
		return "", err
	}
	options := []oauth2.AuthCodeOption{coreoidc.Nonce(nonce)}
	if provider == "slack" {
		options = append(options, oauth2.SetAuthURLParam("team", team))
	}
	return cfg.AuthCodeURL(state, options...), nil
}

func (c *PagingClient) ExchangePaging(ctx context.Context, provider, code, nonce string) (PagingIdentity, error) {
	discovered, cfg, err := c.provider(ctx, provider)
	if err != nil {
		return PagingIdentity{}, err
	}
	token, err := cfg.Exchange(coreoidc.ClientContext(ctx, c.client), code)
	if err != nil {
		return PagingIdentity{}, err
	}
	raw, ok := token.Extra("id_token").(string)
	if !ok || raw == "" {
		return PagingIdentity{}, fmt.Errorf("missing id_token")
	}
	verifier := discovered.VerifierContext(coreoidc.ClientContext(context.Background(), c.client), &coreoidc.Config{ClientID: cfg.ClientID})
	verified, err := verifier.Verify(ctx, raw)
	if err != nil {
		return PagingIdentity{}, err
	}
	if nonce == "" || verified.Nonce != nonce || verified.Subject == "" || verified.Subject == "unknown" {
		return PagingIdentity{}, fmt.Errorf("invalid nonce or subject")
	}
	var claims struct {
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		SlackUserID   string `json:"https://slack.com/user_id"`
		SlackTeamID   string `json:"https://slack.com/team_id"`
	}
	if err := verified.Claims(&claims); err != nil {
		return PagingIdentity{}, err
	}
	return PagingIdentity{Subject: verified.Subject, Email: claims.Email, EmailVerified: claims.EmailVerified,
		SlackUserID: claims.SlackUserID, SlackTeamID: claims.SlackTeamID}, nil
}
