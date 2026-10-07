package oidc

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/aegis/aegis/pkg/config"
	"github.com/stretchr/testify/require"
)

func signedPagingToken(t *testing.T, key *rsa.PrivateKey, claims map[string]any) string {
	t.Helper()
	header, _ := json.Marshal(map[string]string{"alg": "RS256", "kid": "test", "typ": "JWT"})
	payload, _ := json.Marshal(claims)
	unsigned := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	hash := sha256.Sum256([]byte(unsigned))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, hash[:])
	require.NoError(t, err)
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(signature)
}

func TestPagingOIDCDiscoveryAndVerification(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	differentKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	var token string
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			_ = json.NewEncoder(w).Encode(map[string]any{"issuer": server.URL, "authorization_endpoint": server.URL + "/authorize",
				"token_endpoint": server.URL + "/token", "jwks_uri": server.URL + "/jwks", "id_token_signing_alg_values_supported": []string{"RS256"}})
		case "/jwks":
			_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{map[string]string{"kty": "RSA", "use": "sig", "alg": "RS256", "kid": "test",
				"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()), "e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes())}}})
		case "/token":
			require.NoError(t, r.ParseForm())
			require.Equal(t, "test-code", r.Form.Get("code"))
			_ = json.NewEncoder(w).Encode(map[string]string{"access_token": "access", "token_type": "Bearer", "id_token": token})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	cfg := &config.Config{OIDC: map[string]config.OIDCProvider{"slack": {ClientID: "client", ClientSecret: "secret", RedirectURL: "http://aegis.test/auth/slack/callback", Issuer: server.URL},
		"express": {ClientID: "client", ClientSecret: "secret", RedirectURL: "http://aegis.test/auth/express/callback", Issuer: server.URL}}}
	for _, provider := range []string{"slack", "express"} {
		t.Run(provider, func(t *testing.T) {
			client := NewPagingClient(cfg)
			authorization, err := client.AuthorizationURL(context.Background(), provider, "paging.state", "nonce", "T123")
			require.NoError(t, err)
			parsed, err := url.Parse(authorization)
			require.NoError(t, err)
			require.Equal(t, "/authorize", parsed.Path)
			require.Equal(t, "nonce", parsed.Query().Get("nonce"))
			require.Equal(t, "paging.state", parsed.Query().Get("state"))
			if provider == "slack" {
				require.Equal(t, "T123", parsed.Query().Get("team"))
			}
			for _, test := range []struct {
				name       string
				mutate     func(map[string]any)
				signingKey *rsa.PrivateKey
				valid      bool
			}{
				{name: "valid", valid: true},
				{name: "issuer", mutate: func(c map[string]any) { c["iss"] = "https://wrong.test" }},
				{name: "audience", mutate: func(c map[string]any) { c["aud"] = "wrong" }},
				{name: "expired", mutate: func(c map[string]any) { c["exp"] = time.Now().Add(-time.Hour).Unix() }},
				{name: "nonce", mutate: func(c map[string]any) { c["nonce"] = "wrong" }},
				{name: "missing subject", mutate: func(c map[string]any) { delete(c, "sub") }},
				{name: "unknown subject", mutate: func(c map[string]any) { c["sub"] = "unknown" }},
				{name: "signature", signingKey: differentKey},
			} {
				t.Run(test.name, func(t *testing.T) {
					claims := map[string]any{"iss": server.URL, "aud": "client", "sub": "U123", "exp": time.Now().Add(time.Minute).Unix(),
						"iat": time.Now().Unix(), "nonce": "nonce", "email": "alice@example.com", "email_verified": true,
						"https://slack.com/user_id": "U123", "https://slack.com/team_id": "T123"}
					if test.mutate != nil {
						test.mutate(claims)
					}
					signingKey := key
					if test.signingKey != nil {
						signingKey = test.signingKey
					}
					token = signedPagingToken(t, signingKey, claims)
					info, err := client.ExchangePaging(context.Background(), provider, "test-code", "nonce")
					if !test.valid {
						require.Error(t, err)
						return
					}
					require.NoError(t, err)
					require.Equal(t, "U123", info.SlackUserID)
					require.True(t, info.EmailVerified)
				})
			}
			token = ""
			_, err = client.ExchangePaging(context.Background(), provider, "test-code", "nonce")
			require.Error(t, err)
		})
	}
}
