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

1. User chooses provider on login page.
2. `GET /auth/{provider}/login` generates state/nonce, redirects to IdP.
3. Callback validates state, exchanges code for tokens, fetches userinfo.
4. Upsert `users` on `(provider, provider_sub)`.
5. Create `sessions` row; set **HttpOnly**, **Secure** (prod), **SameSite=Lax** cookie.
6. `POST /auth/logout` deletes session server-side.

### Session

- Random session ID stored hashed in `sessions`.
- TTL configurable (default 7 days sliding).
- Middleware rejects expired or missing session on protected routes.

### Slack dual credentials

- **OIDC client** — human login to Aegis UI.
- **Bot token** — outbound pages only. Never used for session establishment.

## Authorization (REQ-AUTH-04)

| Role | Capabilities |
|------|----------------|
| `admin` | Integrations, teams, schedules, users/roles |
| `member` | Ack/resolve/handoff incidents, view all |
| `viewer` | Read-only |

Enforced in service layer + handler checks.

## Secrets (REQ-AUTH-05, NFR-4)

- All tokens in env or `integrations.config` — not in git.
- `.env.example` lists keys with empty values.
- Logs redact `Authorization`, tokens, webhook secrets.

## Webhook security

- Alert webhook: shared secret header `X-Aegis-Webhook-Secret` or HMAC body signature.
- Slack: verify `X-Slack-Signature` timestamp + signing secret.
- eXpress: HMAC per BotX documentation.

## Audit (REQ-AUDIT-01)

Append to `audit_log`:

- Login success/failure (provider, no token)
- Role changes
- Integration create/update/delete
- Routing rule changes

## References

- Setup: [`07-setup-deployment.md`](./07-setup-deployment.md)
- API auth routes: [`04-api-spec.md`](./04-api-spec.md)
