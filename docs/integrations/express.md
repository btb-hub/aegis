# Integration: eXpress

Chat provider and **OIDC sign-in provider**. Uses BotX HTTP API directly (Go `net/http` — no Python SDK).

## Config

### OIDC (auth)

- `EXPRESS_OIDC_ISSUER`, client ID/secret, redirect URI per eXpress SSO docs

### Bot (paging)

- `bot_id`, `host` (BotX CTS URL), `secret_key` for HMAC
- Webhook URL registered with BotX for inbound events

## Identity bootstrap

- Users run `/link` in eXpress bot to bind `express_user_huid` to Aegis user.
- Wizard explains step; admin can revoke links.

## Outbound page

- Send bubble with incident summary + ack button (callback data includes `incident_id`).
- Bubble text and button label from `pkg/i18n` using recipient `users.locale`.

## Inbound ack

- `POST /callbacks/express/bot`
- Verify HMAC per BotX spec
- Parse button action → acknowledge incident

## Test connection

- Health/ping endpoint or lightweight BotX API call with configured credentials.

## Tests

- Recorded BotX payloads in `apps/worker/testdata/express/`.

## References

- REQ-INT-04, REQ-AUTH-01
