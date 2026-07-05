package alertsim

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// SendResult is returned after a successful webhook POST.
type SendResult struct {
	StatusCode int
	Body       string
}

// Client posts alert payloads to the Aegis webhook.
type Client struct {
	URL        string
	Secret     string
	HTTPClient *http.Client
}

// NewClient builds a webhook client. URL should be the full webhook path
// (e.g. http://localhost:8080/api/v1/alerts/webhook).
func NewClient(url, secret string) *Client {
	return &Client{
		URL:    url,
		Secret: secret,
		HTTPClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// Send posts raw JSON to the alert webhook.
func (c *Client) Send(ctx context.Context, raw []byte) (SendResult, error) {
	if c.URL == "" {
		return SendResult{}, fmt.Errorf("webhook URL is required")
	}
	if c.Secret == "" {
		return SendResult{}, fmt.Errorf("webhook secret is required")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.URL, bytes.NewReader(raw))
	if err != nil {
		return SendResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Aegis-Webhook-Secret", c.Secret)

	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return SendResult{}, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	result := SendResult{StatusCode: resp.StatusCode, Body: string(body)}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return result, fmt.Errorf("webhook returned %d: %s", resp.StatusCode, result.Body)
	}
	return result, nil
}

// SendScenario builds and sends a scenario payload.
func (c *Client) SendScenario(ctx context.Context, scenario Scenario, defaults LabelDefaults, suffix string) (SendResult, error) {
	payload := BuildPayload(scenario, defaults, suffix)
	raw, err := MarshalPayload(payload)
	if err != nil {
		return SendResult{}, err
	}
	return c.Send(ctx, raw)
}
