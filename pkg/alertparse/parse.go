package alertparse

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type Payload struct {
	Status      string            `json:"status"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	StartsAt    string            `json:"startsAt"`
}

type ParsedAlert struct {
	Status      string
	Severity    string
	Title       string
	Body        string
	Labels      map[string]string
	Fingerprint string
	Raw         json.RawMessage
}

func Parse(raw json.RawMessage) (*ParsedAlert, error) {
	var payload Payload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("invalid json: %w", err)
	}

	status := strings.ToLower(strings.TrimSpace(payload.Status))
	if status == "" {
		status = "firing"
	}
	if status != "firing" && status != "resolved" {
		return nil, fmt.Errorf("invalid status: %s", payload.Status)
	}

	labels := payload.Labels
	if labels == nil {
		labels = map[string]string{}
	}

	title := payload.Annotations["summary"]
	if title == "" {
		title = labels["alertname"]
	}
	if title == "" {
		title = "Alert"
	}

	body := payload.Annotations["description"]
	severity := labels["severity"]
	if severity == "" {
		severity = "unknown"
	}

	return &ParsedAlert{
		Status:      status,
		Severity:    severity,
		Title:       title,
		Body:        body,
		Labels:      labels,
		Fingerprint: Fingerprint(labels),
		Raw:         raw,
	}, nil
}

func Fingerprint(labels map[string]string) string {
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(labels[k])
		b.WriteByte('|')
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

func ValidateWebhookSecret(provided, expected string) bool {
	if provided == "" || expected == "" {
		return false
	}
	return provided == expected
}
