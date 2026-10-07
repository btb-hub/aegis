# Integration: Slack

Chat provider. Implements `ChatProvider`. Also an **OIDC sign-in provider** (separate credentials).

## Config

### OIDC (auth) — see [`09-security.md`](../09-security.md)

- `SLACK_OIDC_CLIENT_ID`, `SLACK_OIDC_CLIENT_SECRET`, redirect URI

### Bot (paging)

An administrator opens **Integrations → Configure Slack** and saves:

- `bot_token` — the installed app's **Bot User OAuth Token** (`xoxb-...`) from **OAuth & Permissions**.
- `signing_secret` — the app's **Signing Secret** from **Basic Information**.

Grant the bot `chat:write`, install the app in the Slack workspace, and invite it to channels
used for team announcements. On-call recipients need a linked Slack user ID.

Every Aegis workspace has a Slack slot. **Inherit** uses the global bot; its editor links to
**Configure global Slack**. **Custom** saves independent bot credentials for that workspace.
Teams use their workspace's bot and their own Slack channel/user-group settings.

In the Slack app, enable **Interactivity** and set the Request URL to:

`https://<public-aegis-origin>/api/v1/callbacks/slack/interactive`

Slack must be able to reach this URL over HTTPS. The settings form displays the URL using
the browser's public origin. Bot setup is separate from Sign in with Slack.

Paging and callback credentials come from saved integration rows. Setting `SLACK_BOT_TOKEN`
or `SLACK_SIGNING_SECRET` alone does not configure the bot. Saved secret changes take effect
without restarting the API.

## Outbound page

Block Kit message (text from `pkg/i18n` using recipient `users.locale`):

- Header: incident severity + title
- Section: summary, team, link to Aegis
- Actions: **Acknowledge** button (`action_id: ack_incident`) — label translated

Post to DM using `slack_user_id` on the user row. Users can connect, replace, and disconnect Slack in **Account → Paging messengers** through browser authorization. The authorized workspace must match the global paging bot's workspace. This doesn't add a sign-in provider. See [paging connections](../features/paging-connections.md) for setup and API details.

## Contact deep links (AEG-107)

When `users.slack_user_id` is set, current on-call and incident-detail assignee JSON include:

`https://slack.com/app_redirect?channel={slack_user_id}`

This opens Slack for the signed-in user without a request-path Slack API call. Native `slack://user?team={T}&id={U}` is not used until a workspace team ID is stored.

## Inbound ack

- `POST /api/v1/callbacks/slack/interactive`
- Parse the acknowledgement value only to identify the incident and its team's workspace.
- Resolve that workspace's enabled Slack slot using Inherit/Custom rules.
- Verify `X-Slack-Signature` and timestamp against the original request body and resolved secret.
- Only then look up the linked Slack user and acknowledge the incident.

Missing, disabled, or incomplete configurations are rejected. Custom slots never fall back to
other workspaces or environment secrets. Rotating a saved signing secret immediately rejects
signatures made with the old secret.

## Test connection

- `auth.test` API with bot token (admin-only). This checks token authentication; test message
  delivery, channel access, and Acknowledge buttons separately in Slack.

## Tests

- Fixtures for Block Kit payload + signature in `apps/worker/testdata/slack/`.

## References

- REQ-INT-03, REQ-AUTH-01
