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
- `oncall_group_chat_id` — optional global eXpress group chat ID; the sole eXpress destination for on-call announcements

BotX webhook **base** (paste this in the BotX admin; CTS appends `/status` and `/command`):

`{PUBLIC_URL}/api/v1/callbacks/express`

If the BotX admin already has `{PUBLIC_URL}/api/v1/callbacks/express/bot`, that is also a valid base — CTS then calls `…/bot/status` and `…/bot/command`.

| Method | Path | Auth | Response |
|--------|------|------|----------|
| GET | `{base}/status` | BotX JWT optional | **200** `{status:"ok", result:{enabled, status_message, commands}}` listing `/link`, `/ack_incident`, and `/oncall`. Missing integration: **503** `{reason:"bot_disabled", …}`. Invalid JWT: **401** |
| POST | `{base}/command` | BotX JWT | **202** `{"result":"accepted"}` (≤5s). Command outcomes and reply jobs commit together, deduplicated by command ID. No outbound requests occur in the callback. Business failures also return 202 with a separately queued localized reply; failures before durable acceptance return retryable errors. System events are silent |
| POST | `{base}/bot` | BotX JWT | Deprecated alias of `/command` |

Must not sit behind interactive Google/IAP login.

## Identity bootstrap (`/link`)

1. User signs in to Aegis and calls `POST /api/v1/users/me/express-link-code`.
2. Response includes a short-lived code and the command to send in eXpress: `/link <code>`.
3. User sends that command to the Aegis bot in eXpress.
4. BotX delivers the command to `{base}/command`; Aegis binds `express_user_huid` to the user.

**Direct bind stub (admin/testing):** `POST /api/v1/users/me/express-link` with body
`{"express_user_huid":"<uuid>"}` binds the huid without the bot flow.

Codes are stored in `express_link_codes` and expire after 15 minutes.

## Contact deep links (AEG-107)

When `users.express_user_huid` is set, current on-call and incident-detail assignee JSON include:

`https://xlnk.ms/open/profile/{express_user_huid}`

That is the documented eXpress user-contact link. The client can start a DM from the profile.

## Outbound page

- `POST /api/v4/botx/notifications/direct` with bubble ack button (`/ack_incident` + `incident_id` in `data`).
- Resolve the linked user’s personal bot chat via `GET /api/v1/botx/chats/personal?user_huid=…`, then supply the non-null `group_chat_id`. Existing users need not relink. If no chat exists, the engineer must open the bot chat; the shared channel is never a private-page fallback.
- Bubble text and button label from `pkg/i18n` using recipient `users.locale`.
- Bot token obtained via `GET /api/v2/botx/bots/{bot_id}/token?signature=<HMAC-SHA256(bot_id)>`.

## Outbound on-call announce

- `POST /api/v4/botx/notifications/direct` with `group_chat_id` = global `oncall_group_chat_id`.
- `teams.express_chat_id` is retained for rollback compatibility but is not used for on-call publication.
- Body mentions on-call users with `@{mention:<mention_id>}` and `mentions[]` (`mention_type: "user"`, `mention_data.user_huid`). Users without a huid are listed by name only.
- Worker job `publish_oncall` (daily, rotation change, or `POST /api/v1/teams/{id}/on-call/publish`). Do not reuse incident DM `SendPage`.

## Inbound ack and link

- `GET /api/v1/callbacks/express/status` and `GET /api/v1/callbacks/express/bot/status` — bot alive + command list (`/link`, `/ack_incident`, `/oncall`)
- `POST /api/v1/callbacks/express/command` and `POST /api/v1/callbacks/express/bot/command` — commands (`/link`, ack, system events)
- `POST /api/v1/callbacks/express/bot` — deprecated alias of `/command`
- Verify BotX JWT in `Authorization` header (HS256, `secret_key`) on `/command`. Status verifies JWT when the header is present.
- `/link <code>` → bind huid, then 202
- `/ack_incident` (or bubble `data.incident_id`) → acknowledge incident, then 202

## Test connection

- Obtains a BotX token using configured `bot_id` and `secret_key` (same as paging).

## Tests

- Recorded BotX payloads in `apps/worker/testdata/express/`.
- Provider tests in `pkg/integrations/express/`.

## References

- REQ-INT-04, REQ-INT-05, REQ-AUTH-01

## Shared incidents, replies and delivery results

Each new incident posts once to the global `oncall_group_chat_id`, with an incident link and Acknowledge button. Assigned engineers retain private pages, including handoff/escalation events, using workspace settings. Deliveries are tracked separately by connector, destination and event; retrying one does not resend another accepted delivery.

The `/oncall` command returns every team's current engineers and actual shift ranges in Europe/Moscow to the incoming chat. Linking is unnecessary for this read-only command. Announcement mentions retain linked users and name-only fallback. Daily publication is configurable by administrators in Settings, initially **03:00 UTC**; changes take effect within one minute. See [API settings](../04-api-spec.md).

Register `POST {base}/notification/callback` (also supported at `…/bot/notification/callback`) for BotX asynchronous results. Requests require a valid enabled bot signature. Aegis correlates the returned `sync_id`, retains an inbox for callbacks arriving before outgoing persistence, and deduplicates repeated results. Incident failures appear on the timeline; reply failures remain in `express_outbound`, and announcement failures in `oncall_deliveries`. An HTTP 202 from `/direct` means accepted, not confirmed delivered.

Command replies are localized in English/Russian and report linked, acknowledged, already acknowledged, missing/resolved incident, unlinked identity, invalid/expired link code, unsupported command or denied permissions. Long `/oncall` replies split into individually tracked messages, and retries resume unsent parts.

Only global eXpress configuration controls shared announcements/channel incident posts. Workspace settings control private pages; `teams.express_chat_id` remains unused for announcements. Restore/configure the global connector through Integrations after release; no installation-specific IDs belong in code.

[BotX API documentation](https://hackmd.ccsteam.ru/s/E9MPeOxjP)
