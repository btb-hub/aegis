# Integration: Slack

Chat provider. Implements `ChatProvider`. Also an **OIDC sign-in provider** (separate credentials).

## Config

### OIDC (auth) — see [`09-security.md`](../09-security.md)

- `SLACK_OIDC_CLIENT_ID`, `SLACK_OIDC_CLIENT_SECRET`, redirect URI

### Bot (paging)

- `bot_token` — `xoxb-...`
- `signing_secret` — interactive payload verification

## Outbound page

Block Kit message (text from `pkg/i18n` using recipient `users.locale`):

- Header: incident severity + title
- Section: summary, team, link to Aegis
- Actions: **Acknowledge** button (`action_id: ack_incident`) — label translated

Post to DM using `slack_user_id` on user row (mapped at login or admin link).

## Inbound ack

- `POST /callbacks/slack/interactive`
- Verify `X-Slack-Signature`
- Parse `action_id`, `incident_id` from value
- Call incident acknowledge service

## Test connection

- `auth.test` API with bot token.

## Tests

- Fixtures for Block Kit payload + signature in `apps/worker/testdata/slack/`.

## References

- REQ-INT-03, REQ-AUTH-01
