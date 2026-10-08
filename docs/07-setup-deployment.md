# Deployment and settings

Aegis runs API, worker and React UI with PostgreSQL 16. Application settings are stored in
PostgreSQL and managed by admins at **Settings / Настройки**. The setup wizard is removed.

## Production image (GHCR)

For an existing installation, follow [the manual configuration migration](configuration-migration.md)
before starting updated services. Back up the database and preserve the original env first.
Apply migrations, dry-run against the actual production file, apply the import, deploy, verify,
and only then remove migrated variables. Leave additive tables in place for binary rollback.

For a fresh installation, generate a token from at least 32 random bytes and keep it secret:

```bash
openssl rand -hex 32
```

Store the generated token and database connection in protected deployment configuration:

```dotenv
DATABASE_URL=postgres://user:pass@db-host:5432/aegis?sslmode=require
AEGIS_BOOTSTRAP_TOKEN=<generated-token>
DEV_AUTH_ENABLED=false
```

```bash
docker pull ghcr.io/btb-hub/aegis:vX.Y.Z
docker run --rm -p 3000:3000 --env-file /secure/deployment.env ghcr.io/btb-hub/aegis:vX.Y.Z
```

Run one replica. The image applies migrations and starts API, worker and nginx. The container
serves the UI, `/healthz`, `/readyz`, and `/metrics` on port 3000. Never expose PostgreSQL publicly.

Open the app's public HTTPS URL, then `/bootstrap`. Submit the token through the form, enter the
first admin email, save provider credentials, register the displayed callback externally and
complete **Test sign-in** with that verified email. This atomically activates the provider,
creates the first admin/session and permanently closes bootstrap. Remove the deployment token.
Existing installations cannot use bootstrap. See the [settings guide](features/settings.md).

### Kubernetes sketch

Use [deploy/k8s/aegis.yaml](../deploy/k8s/aegis.yaml) with Secret `aegis-env` containing only
deployment keys. Point ingress at Service port 80. The bootstrap form saves the public origin;
subsequent changes are made under Settings. Keep one replica. Helm remains out of scope.

### Outer identity proxy

Aegis does not ship an interactive identity-aware proxy. oauth2-proxy and Google IAP are optional
ops layers in front of the UI. They are separate Google logins from Aegis OIDC, and they are not
the same product: IAP uses GCP cookies and BackendConfig path exemptions, not `/oauth2/callback`.

| Layer | Callback | Typical failure |
|-------|----------|-----------------|
| oauth2-proxy | `/oauth2/callback` | HTML 403 with footer "Secured with OAuth2 Proxy" |
| Aegis OIDC | `/auth/{provider}/callback` | JSON 400; cookie `aegis_oauth_state` |

The proxy does not replace Aegis OIDC. After the outer gate, humans still sign in with Google,
Slack, or eXpress on the Aegis login page. Register a **second** Google OAuth client (or a second
authorized redirect URI) for `https://<ui-host>/oauth2/callback`. Do not point any Aegis
`*_OIDC_REDIRECT_URL` at `/oauth2/callback`.

If a normal browser profile fails instantly and Incognito succeeds, or if both fail with the proxy
403 page, fix the proxy.

Two competing diagnoses for “Incognito works”:

1. **Stale `_oauth2_proxy` cookies.** Delete every cookie for the host whose name starts with
   `_oauth2_proxy`. Clicking Sign in on the 403 page is not enough. Expand **More Info** if you see
   CSRF token missing or mismatch.
2. **Wrong Google account.** Chrome reused a personal Gmail session outside the proxy email
   allowlist (`email_domains` / `--email-domain`). Expand **More Info** for "email not authorized".
   Use the Google account chooser, or set `prompt = "select_account"` in the proxy config.

Copy [`deploy/oauth2-proxy.example.cfg`](../deploy/oauth2-proxy.example.cfg) (oauth2-proxy v7.15.3).
Set CSRF-per-request, SameSite Lax, and an **exact-host** cookie domain that matches `redirect_url`
(or omit `cookie_domains` for a host-only cookie). A parent domain such as `.example.com` leaves
duplicate CSRF cookies in Chrome and reproduces the Incognito-only success.

Keep `/api/v1/callbacks/`, `/api/v1/alerts/webhook`, `/healthz`, `/readyz`, and `/metrics` off the
interactive proxy (REQ-INT-07: path skip or a second public host). Kube probes should hit the
in-cluster Service, not the public proxy host. Do not leave `/metrics` reachable on the public
internet without a network policy.

### Publishing (maintainers)

Push a version tag (`v0.1.0`, …). GitHub Actions builds `linux/amd64` + `linux/arm64` and pushes
`ghcr.io/btb-hub/aegis:<tag>` and `:latest`. After the first push, set the GHCR package visibility
to **Public** in the GitHub UI (one-time).

## Local development

Prerequisites: Docker with Compose v2, Go 1.25+, Node 20+, and Make (or `scripts/dev.ps1`).

```bash
make setup-local
make dev-db
make dev-api
make dev-worker
make dev-web
```

Run each dev process in its own terminal. The local template enables dev sign-in and uses
`http://localhost:3000` with the API on `:8080`. Don't enable dev auth in production. For the
full Docker stack, use `make setup`, set `DEV_AUTH_ENABLED=true` for local testing, and `make up`.
Sign in once before `make seed-dev`, or explicitly import a legacy local env before startup.
The seed tool retains its development safeguards (`SEED_DEV`); tool variables aren't app settings.

## Environment variables

| Key | Purpose |
| --- | --- |
| `DATABASE_URL` | Required PostgreSQL connection |
| `AEGIS_BOOTSTRAP_TOKEN` | Fresh installations only; generated from at least 32 random bytes |
| `DEV_AUTH_ENABLED` | Localhost sign-in only; default false in production |
| `DEV_AUTH_DEFAULT_ROLE` | Local dev role: admin, member or viewer |
| `DEV_AUTH_EMAIL` | Local dev user's email |
| `SEED_DEV` and `AEGIS_*` process/tool overrides | Tooling only |

Templates: [Docker](../deploy/.env.example), [native local](../deploy/.env.local.example).
The [migration manual](configuration-migration.md) maps every legacy application key.
Authentication credentials, admin email rules, bots, public URL, webhook secret, listening
address and behavior settings are managed in PostgreSQL. API/worker ignore their old env values.
`SESSION_SECRET` is kept in compatibility storage without a runtime consumer.

## Configure the workflow

1. Sign in as admin and configure **Settings → Sign-in and access**. Save/test provider drafts
   before activation. Manage admin email rules here; they apply on subsequent verified logins.
2. Save, enable and test Jira/Slack/eXpress under **Settings → Integrations**. Workspace slots
   remain on workspace pages. Slack interactivity uses `/api/v1/callbacks/slack/interactive`.
3. Create workspaces, teams, membership, escalation paths, schedules and routing on their
   existing pages. See the [workflow guide](how-to/setup.md).
4. Connect each responder's personal messengers on **Account**.
5. Use **Settings → Diagnostics** for health and a test alert. The new endpoint is
   `POST /api/v1/settings/test-alert`; `/api/v1/setup/test-alert` remains a compatibility alias.

`/setup` redirects to Settings. `/integrations` redirects to its Integrations section,
retaining configuration query parameters. There is no wizard or browser progress storage.

## Operations

| Concern | Action |
| --- | --- |
| Health/readiness | `/healthz`, `/readyz`; startup refuses existing installs without import |
| Backups | Protect database backups as secret storage; test restore |
| App and connector secret rotation | Update Settings; blank secret inputs retain values |
| Listener change | Save in Deployment, restart API, match nginx upstream configuration |
| Public URL change | Confirm displayed callbacks, register them externally, pending authorizations expire |
| Session/fingerprint/escalation change | Applies to subsequent operations; stored sessions, alerts and jobs are preserved |
| Rollback | Restore old binaries/env, leave additive settings tables in place |
| Deployment secret change | Update protected env/secret store and recreate services |
| Metrics | Restrict `/metrics` with proxy/network policy |

The local simulator remains separate development tooling. Configure its API URL and webhook
secret separately, using the webhook secret saved in Settings. Never run it in production.

## Migrations and verification

```bash
make migrate-up
make lint type test
```

Existing production upgrades also require the explicit configuration importer between schema
migration and service deployment. Do not reverse `000022` during a binary rollback. Its down
migration removes installation state and settings; use it only in a deliberate schema rollback.
