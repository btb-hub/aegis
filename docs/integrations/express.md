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

BotX webhook **base** (paste this in the BotX admin; CTS appends paths):

`{PUBLIC_URL}/api/v1/callbacks/express`

| Method | Path | Auth | Response |
|--------|------|------|----------|
| GET | `{base}/status` | BotX JWT (`Authorization`) | **200** `{status:"ok", result:{enabled, status_message, commands}}`. Missing integration: **503** `{reason:"bot_disabled", …}` |
| POST | `{base}/command` | BotX JWT | **202** `{"result":"accepted"}` (≤5s). `/link` and ack are processed synchronously, then accepted. `system:*` and unknown commands also 202 |
| POST | `{base}/bot` | BotX JWT | Deprecated alias of `/command` |

Must not sit behind interactive Google/IAP login.

## Identity bootstrap (`/link`)

1. User signs in to Aegis and calls `POST /api/v1/users/me/express-link-code`.
2. Response includes a short-lived code and the command to send in eXpress: `/link <code>`.
3. User sends that command to the Aegis bot in eXpress.
4. BotX delivers the command to `/callbacks/express/command`; Aegis binds `express_user_huid` to the user.

**Direct bind stub (admin/testing):** `POST /api/v1/users/me/express-link` with body
`{"express_user_huid":"<uuid>"}` binds the huid without the bot flow.

Codes are stored in `express_link_codes` and expire after 15 minutes.

## Outbound page

- `POST /api/v4/botx/notifications/direct` with bubble ack button (`/ack_incident` + `incident_id` in `data`).
- Bubble text and button label from `pkg/i18n` using recipient `users.locale`.
- Bot token obtained via `GET /api/v2/botx/bots/{bot_id}/token?signature=<HMAC-SHA256(bot_id)>`.

## Outbound on-call announce

- `POST /api/v4/botx/notifications` with `group_chat_id` = `teams.express_chat_id`.
- Body mentions on-call users with `@{mention:<mention_id>}` and `mentions[]` (`mention_type: "user"`, `mention_data.user_huid`). Users without a huid are listed by name only.
- Worker job `publish_oncall` (daily, rotation change, or `POST /api/v1/teams/{id}/on-call/publish`). Do not reuse incident DM `SendPage`.

## Inbound ack and link

- `GET /api/v1/callbacks/express/status` — bot alive + command list (`/link`)
- `POST /api/v1/callbacks/express/command` — commands (`/link`, ack, system events)
- `POST /api/v1/callbacks/express/bot` — deprecated alias of `/command`
- Verify BotX JWT in `Authorization` header (HS256, `secret_key`)
- `/link <code>` → bind huid, then 202
- `/ack_incident` (or bubble `data.incident_id`) → acknowledge incident, then 202

## Test connection

- Obtains a BotX token using configured `bot_id` and `secret_key` (same as paging).

## Tests

- Recorded BotX payloads in `apps/worker/testdata/express/`.
- Provider tests in `pkg/integrations/express/`.

## References

- REQ-INT-04, REQ-INT-05, REQ-AUTH-01
