# Paging messenger connections

Open **Account → Paging messengers** to connect Slack or eXpress. Connect and Replace open browser authorization and return to Account. No bot command or manually entered messenger ID is required. Replace keeps the current connection until the new authorization succeeds. Disconnect requires confirmation and stops paging through that messenger.

Paging connections and sign-in providers are separate. Authorizing a paging connection changes only the current user's paging identity. It doesn't create an Aegis user, merge accounts by email, change the session, or add a sign-in provider. Disconnect does not remove sign-in identities. An explicitly managed Slack connection is never backfilled by a later login, even after Disconnect.

## Administrator setup

- Apply migrations through `000022_application_settings` and complete the [configuration import](../configuration-migration.md) on existing installations before deploying the API and worker.
- In **Settings → Sign-in and access**, save and test Slack or eXpress credentials before activating the provider. eXpress also needs its company SSO issuer. Imported credentials and exact callback URLs are preserved.
- Register the existing callbacks with the provider: `{PUBLIC_URL}/auth/slack/callback` and `{PUBLIC_URL}/auth/express/callback`. Sign-in and paging share these registered URLs; paging attempts are dispatched separately through session-bound state.
- The issuer must expose OpenID discovery and signing keys. Authorization uses `openid email profile`; returned ID tokens must have a valid signature, issuer, client audience, expiry, subject, and matching nonce. Browser authorization must return via GET with the existing Aegis session cookie.
- Configure and enable the global Slack or eXpress bot under **Settings → Integrations**. Account status reflects global bot configuration, not a workspace override. The public URL in **Settings → Deployment** must match the browser-facing scheme and host so mutation origin checks work behind proxies.
- **Slack:** use Sign in with Slack credentials, separate from bot OAuth scopes. The bot must have access to its paging workspace. Aegis calls `auth.test` and requires the authorized `https://slack.com/team_id` to match that workspace. It uses the verified `https://slack.com/user_id` for paging. See [Slack authorization](https://docs.slack.dev/authentication/sign-in-with-slack).
- **eXpress:** configure company SSO to include the actual user's email and boolean `email_verified: true` in the signed ID token. The email must match the messenger directory. BotX must allow `POST /api/v3/botx/users/by_email` using the existing bot credentials. Lookup is local to the configured CTS, without trust-server search. Aegis requires exactly one distinct active HUID with an exact, case-insensitive email match. SSO `sub` and directory `user_id` aren't treated as messenger HUIDs. See [BotX user lookup](https://docs.express.ms/chatbots/developer-guide/api/botx-api/users-api).

Missing configuration disables Connect and Replace. Account distinguishes missing authentication, missing bot credentials, and a disabled bot, with Settings links for administrators. Already connected users can still Disconnect. Discovery, bot permission, workspace, and email lookup failures preserve the current connection and show an actionable error.

## API

All endpoints require the Aegis session. POST and DELETE additionally require a same-origin `Origin` or `Referer`; cross-site requests are rejected.

| Method | Path | Result |
| --- | --- | --- |
| GET | `/api/v1/users/me/paging-connections` | `{ "connections": [{ "provider": "slack", "identity": "U123", "available": true }, { "provider": "express", "identity": null, "available": false, "unavailable_reason": "authentication_required" }] }` |
| POST | `/api/v1/users/me/paging-connections/{provider}/authorize` | `{ "authorization_url": "https://..." }`; no request body |
| DELETE | `/api/v1/users/me/paging-connections/{provider}` | `204`; no request body |

Only `slack` and `express` are supported. Callback results redirect to `/account?paging_result=connected&paging_provider=...` or `/account?paging_error=...&paging_provider=...`. Errors contain fixed codes, never upstream response bodies or credentials. Account consumes these parameters and refreshes session data; the callback doesn't set a new session cookie.

## Persistence and compatibility

Authorizations expire after five minutes and are bound to the initiating user, session, provider, and nonce. State is stored as a hash. Each callback is claimed once before exchanging with the provider; Disconnect and a new authorization invalidate in-flight attempts. Session deletion cascades to pending attempts. Updates, invalidation, and `paging.connected` / `paging.disconnected` audit records commit together.

The existing `users.slack_user_id` and `users.express_user_huid` fields remain the worker and acknowledgment sources. Migration preserves all existing values. All paging writers serialize ownership checks per provider and reject assigning an identity already connected to another account. Existing duplicate values aren't silently removed during migration and may require administrator cleanup.

The eXpress link-code and direct-bind endpoints remain compatible. Successfully connecting, replacing, or disconnecting eXpress invalidates older bot link codes as well as pending browser attempts. Generating a new code remains possible through the API.

## Verification

Automated coverage includes signed-token verification, API and account flows, directory and workspace checks, cross-site rejection, and real PostgreSQL tests for replay, expiry, concurrent ownership, replacement, and durable disconnection.

Before rollout, authorize Slack and eXpress in the configured deployment, confirm the linked account and workspace, send a test incident page through each messenger, and acknowledge it. Repeat after replacement. Disconnect, sign in again, and confirm that the connection remains absent. Automated provider fixtures don't replace this live deployment check.
