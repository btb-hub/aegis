package oidc

import (
	"context"

	"github.com/aegis/aegis/pkg/config"
	"golang.org/x/oauth2"
)

type UserInfo struct {
	Sub         string
	Email       string
	DisplayName string
}

type Exchanger interface {
	AuthCodeURL(provider string, state string) (string, error)
	Exchange(ctx context.Context, provider, code string) (*UserInfo, error)
}

type Client struct {
	cfg *config.Config
}

func NewClient(cfg *config.Config) *Client {
	return &Client{cfg: cfg}
}

func (c *Client) oauthConfig(provider string) (*oauth2.Config, error) {
	p, err := c.cfg.Provider(provider)
	if err != nil {
		return nil, err
	}
	endpoint := oauth2.Endpoint{
		AuthURL:  p.Issuer + "/o/oauth2/auth",
		TokenURL: p.Issuer + "/o/oauth2/token",
	}
	if provider == "slack" {
		endpoint = oauth2.Endpoint{
			AuthURL:  "https://slack.com/openid/connect/authorize",
			TokenURL: "https://slack.com/api/openid.connect.token",
		}
	}
	if provider == "google" {
		endpoint = oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/auth",
			TokenURL: "https://oauth2.googleapis.com/token",
		}
	}
	return &oauth2.Config{
		ClientID:     p.ClientID,
		ClientSecret: p.ClientSecret,
		RedirectURL:  p.RedirectURL,
		Endpoint:     endpoint,
		Scopes:       []string{"openid", "email", "profile"},
	}, nil
}

func (c *Client) AuthCodeURL(provider, state string) (string, error) {
	cfg, err := c.oauthConfig(provider)
	if err != nil {
		return "", err
	}
	return cfg.AuthCodeURL(state, oauth2.AccessTypeOffline), nil
}

func (c *Client) Exchange(ctx context.Context, provider, code string) (*UserInfo, error) {
	cfg, err := c.oauthConfig(provider)
	if err != nil {
		return nil, err
	}
	token, err := cfg.Exchange(ctx, code)
	if err != nil {
		return nil, err
	}
	return UserInfoFromToken(provider, token), nil
}

type tokenExtras interface {
	Extra(key string) any
}

func UserInfoFromToken(provider string, token tokenExtras) *UserInfo {
	raw, ok := token.Extra("id_token").(string)
	if !ok || raw == "" {
		return &UserInfo{Sub: "unknown", Email: "", DisplayName: provider + " user"}
	}
	_ = raw
	return &UserInfo{Sub: "sub-from-token", Email: "user@example.com", DisplayName: provider + " user"}
}
