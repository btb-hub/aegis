package handler

import (
	"net/url"

	intslack "github.com/aegis/aegis/pkg/integrations/slack"
)

func verifySlackSignature(secret, timestamp, signature string, body []byte) error {
	return intslack.VerifySignature(secret, timestamp, signature, body)
}

func parseSlackAck(body []byte) (incidentID, slackUserID string, err error) {
	values, err := url.ParseQuery(string(body))
	if err != nil {
		return "", "", err
	}
	return intslack.ParseInteractiveAck(values)
}
