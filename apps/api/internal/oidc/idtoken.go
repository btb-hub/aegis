package oidc

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

const slackUserIDClaim = "https://slack.com/user_id"

type idTokenClaims struct {
	Sub     string `json:"sub"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
	// Slack OpenID uses namespaced claims; capture via raw map as well.
	SlackUserID string `json:"-"`
}

func ParseIDTokenPayload(idToken string) (map[string]any, error) {
	parts := strings.Split(idToken, ".")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid id_token format")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode id_token payload: %w", err)
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("parse id_token claims: %w", err)
	}
	return claims, nil
}

func UserInfoFromClaims(provider string, claims map[string]any) *UserInfo {
	if claims == nil {
		return &UserInfo{Sub: "unknown", DisplayName: provider + " user"}
	}

	var parsed idTokenClaims
	raw, _ := json.Marshal(claims)
	_ = json.Unmarshal(raw, &parsed)

	if parsed.Sub == "" {
		if sub, ok := claims["sub"].(string); ok {
			parsed.Sub = sub
		}
	}
	if parsed.Email == "" {
		parsed.Email = stringClaim(claims, "email")
	}
	if parsed.Name == "" {
		parsed.Name = firstNonEmpty(
			stringClaim(claims, "name"),
			strings.TrimSpace(stringClaim(claims, "given_name")+" "+stringClaim(claims, "family_name")),
		)
	}
	if parsed.Picture == "" {
		parsed.Picture = stringClaim(claims, "picture")
	}

	info := &UserInfo{
		Sub:         parsed.Sub,
		Email:       parsed.Email,
		DisplayName: parsed.Name,
		AvatarURL:   parsed.Picture,
	}

	if provider == "slack" {
		info.SlackUserID = stringClaim(claims, slackUserIDClaim)
	}

	if info.Sub == "" {
		info.Sub = "unknown"
	}
	if info.DisplayName == "" {
		info.DisplayName = provider + " user"
	}
	return info
}

func stringClaim(claims map[string]any, key string) string {
	value, ok := claims[key]
	if !ok {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
