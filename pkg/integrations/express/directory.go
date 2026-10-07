package express

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// LookupPagingHUID uses the authenticated SSO email, never an arbitrary browser input.
func (p *Provider) LookupPagingHUID(ctx context.Context, email string) (string, error) {
	email = strings.TrimSpace(email)
	if email == "" {
		return "", fmt.Errorf("missing email")
	}
	var huid string
	err := p.withRetry(ctx, func() error {
		token, err := p.ensureToken(ctx)
		if err != nil {
			return err
		}
		body, err := p.postJSON(ctx, "/api/v3/botx/users/by_email", map[string]any{"emails": []string{email}}, token)
		if err != nil {
			return err
		}
		var response struct {
			Status string `json:"status"`
			Result []struct {
				HUID   string   `json:"user_huid"`
				Active bool     `json:"active"`
				Emails []string `json:"emails"`
			} `json:"result"`
		}
		if err := json.Unmarshal(body, &response); err != nil {
			return err
		}
		if response.Status != "ok" {
			return fmt.Errorf("directory lookup failed")
		}
		matches := make(map[string]bool)
		for _, user := range response.Result {
			if !user.Active {
				continue
			}
			for _, candidate := range user.Emails {
				if strings.EqualFold(strings.TrimSpace(candidate), email) {
					id, err := uuid.Parse(user.HUID)
					if err != nil || id == uuid.Nil {
						return fmt.Errorf("invalid directory HUID")
					}
					matches[id.String()] = true
				}
			}
		}
		if len(matches) != 1 {
			return fmt.Errorf("expected exactly one active messenger account matching SSO email")
		}
		for id := range matches {
			huid = id
		}
		return nil
	})
	return huid, err
}
