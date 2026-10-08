# Security

## Authentication

### OIDC only (REQ-AUTH-01, REQ-AUTH-02)

MVP supports exactly three identity providers:

| Provider | Use |
|----------|-----|
| **Google** | Workforce / Google Workspace sign-in |
| **Slack** | Sign in with Slack (OpenID Connect) |
| **eXpress** | Corporate eXpress SSO (OIDC) |

**Not in MVP:** local username/password, Keycloak, LDAP, Authentik, or any self-hosted IdP.

### Flow

1. User chooses provider on the login page ([`docs/features/web-auth.md`](./features/web-auth.md); Phase 3.5).
2. `GET /auth/{provider}/login` generates state/nonce, redirects to IdP. Unknown or unconfigured
   providers redirect to `/login?auth_error=unconfigured&provider={name}` (browser-safe).
3. Callback claims a single-use database authorization bound to the browser, provider, settings
   revision and five-minute expiry. Verify the signed ID token through discovery/JWKS, including
   issuer, audience, expiry, nonce and verified email, before creating or linking a user.
4. Upsert `users` on `(provider, provider_sub)`.
5. Create `sessions` row; set **HttpOnly**, **Secure** (prod), **SameSite=Lax** cookie.
6. `POST /auth/logout` deletes session server-side.

### Session

- Random session ID stored hashed in `sessions`.
- TTL configurable in Settings (default 7 days). Existing session expiry timestamps are preserved.
- Middleware rejects expired or missing session on protected routes.

### Provider changes and first access

Admin provider edits remain drafts until a signed-token test verifies the initiating admin's
email. Tests bind the provider, draft/settings revision, and browser session, expire after
five minutes, and cannot create users or paging links. Activation rejects stale tests and
disabling the last active provider. Public URL changes invalidate pending authorizations.

Fresh installations accept a deployment token generated from at least 32 random bytes through
the bootstrap form, never a URL. Its hashed browser session expires after 30 minutes and only
allows authentication setup. Verified sign-in with the nominated first admin email atomically
creates the admin/session, activates the provider, and permanently closes bootstrap. Existing
installations require the explicit configuration import and cannot enter bootstrap. See
[Settings](./features/settings.md) and the [manual rollout](./configuration-migration.md).

### Outer proxy vs Aegis session

If an identity-aware proxy sits in front of the UI, it is not the Aegis session (`aegis_session`)
or the Aegis OIDC callback (`/auth/{provider}/callback`). oauth2-proxy uses `_oauth2_proxy*` cookies
and `/oauth2/callback`. Google IAP uses GCP cookies and path exemptions, not that callback. A 403
HTML page that says "Secured with OAuth2 Proxy" is the oauth2-proxy gate. Diagnose it in
[`07-setup-deployment.md`](./07-setup-deployment.md#outer-identity-proxy).

### Slack dual credentials

- **OIDC client** — human login to Aegis UI.
- **Bot token** — outbound pages only. Never used for session establishment.

### Paging authorization

Account messenger connections use separate, single-use, five-minute authorizations bound to the initiating Aegis session and user. Signed ID tokens are verified through OIDC discovery and JWKS, including issuer, client audience, expiry, subject, and nonce. Slack workspace identity must match the global bot; eXpress maps verified SSO email through the authenticated BotX directory. Paging callbacks never create users, merge accounts, or replace the session. Browser mutations require same-origin Origin/Referer. See [paging connections](./features/paging-connections.md).

## Authorization (REQ-AUTH-04)

| Role | Capabilities |
|------|----------------|
| `admin` | Integrations, teams, schedules, users/roles |
| `member` | Ack/resolve/handoff incidents, view all |
| `viewer` | Read-only |

Enforced in service layer + handler checks.

- `ADMIN_EMAILS` is a database-backed rule in Settings: matching verified logins are forced to
  `admin`. These users cannot be demoted while pinned. Last-admin protection remains enforced.

## Secrets (REQ-AUTH-05, NFR-4)

- App secrets live in `application_settings`, provider drafts and existing `integrations.config`.
  Database backups and database access must be protected as secret storage.
- Deployment env retains the database connection and first-install token; local dev safeguards
  remain deployment-side and disabled in production.
- Settings responses expose presence flags and empty secret inputs. Audits contain field names
  only. Importer output contains key names/statuses only. Blank secret inputs retain values.
- Logs redact `Authorization`, tokens, webhook secrets.
- `GET /integrations` and `GET /integrations/{id}` redact secret config keys
  (`api_token`, `bot_token`, `signing_secret`, `secret_key`) as `***`. Prefer
  `PATCH /integrations/{id}` with blank secrets to keep stored values rather than
  round-tripping list payloads.

## Webhook security

- Alert webhook: shared secret header `X-Aegis-Webhook-Secret` or HMAC body signature.
- Slack: verify `X-Slack-Signature` timestamp + signing secret.
- eXpress: BotX JWT (HS256, `secret_key`) on `POST /api/v1/callbacks/express/command`
  (and `/bot/command`, deprecated `POST …/bot`). `GET …/status` and `GET …/bot/status`
  verify JWT when `Authorization` is present. These paths are machine ingress
  (REQ-INT-07) and must not sit behind interactive IAP.

## Audit (REQ-AUDIT-01)

Append to `audit_log`:

- Login success/failure (provider, no token)
- Role changes
- Integration create/update/delete
- Routing rule changes

## References

- Setup: [`07-setup-deployment.md`](./07-setup-deployment.md)
- API auth routes: [`04-api-spec.md`](./04-api-spec.md)
