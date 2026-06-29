# Integration: eXpress

Chat provider and **OIDC sign-in provider**. Uses BotX HTTP API directly (Go `net/http` — no Python SDK).

## Config

### OIDC (auth)

- `EXPRESS_OIDC_ISSUER`, client ID/secret, redirect URI per eXpress SSO docs

### Bot (paging)

Integration `config` JSON:

- `bot_id` — BotX bot UUID
- `host` — BotX CTS base URL (e.g. `https://cts.example.com`)
- `secret_key` — bot secret for HMAC token signing and JWT verification

Webhook URL registered with BotX: `POST {PUBLIC_URL}/api/v1/callbacks/express/bot`

## Identity bootstrap (`/link`)

1. User signs in to Aegis and calls `POST /api/v1/users/me/express-link-code`.
2. Response includes a short-lived code and the command to send in eXpress: `/link <code>`.
3. User sends that command to the Aegis bot in eXpress.
4. BotX delivers the command to `/callbacks/express/bot`; Aegis binds `express_user_huid` to the user.

**Direct bind stub (admin/testing):** `POST /api/v1/users/me/express-link` with body
`{"express_user_huid":"<uuid>"}` binds the huid without the bot flow.

Codes are stored in `express_link_codes` and expire after 15 minutes.

## Outbound page

- `POST /api/v4/botx/notifications/direct` with bubble ack button (`/ack_incident` + `incident_id` in `data`).
- Bubble text and button label from `pkg/i18n` using recipient `users.locale`.
- Bot token obtained via `GET /api/v2/botx/bots/{bot_id}/token?signature=<HMAC-SHA256(bot_id)>`.

## Inbound ack and link

- `POST /api/v1/callbacks/express/bot`
- Verify BotX JWT in `Authorization` header (HS256, `secret_key`)
- `/link <code>` → bind huid
- `/ack_incident` (or bubble `data.incident_id`) → acknowledge incident

## Test connection

- Obtains a BotX token using configured `bot_id` and `secret_key` (same as paging).

## Tests

- Recorded BotX payloads in `apps/worker/testdata/express/`.
- Provider tests in `pkg/integrations/express/`.

## References

- REQ-INT-04, REQ-INT-05, REQ-AUTH-01
