# Setup & deployment

MVP runs on **Docker Compose** — Postgres, API, worker, web. No Redis.

## Prerequisites

- Docker 24+ and Docker Compose v2
- Go 1.25+ (local dev)
- Node 20+ (local web dev)

## Quick start

### Full stack (Docker)

```bash
make setup
# Edit .env — SESSION_SECRET, WEBHOOK_SECRET, and OIDC client IDs/secrets
make up
```

Open `http://localhost:3000` (web) and `http://localhost:8080` (API). Stop with `make down`.

`make up` applies database migrations automatically before starting services (Postgres migrate container + explicit `migrate-docker` step).

Equivalent without Make:

```bash
cp deploy/.env.example .env
docker compose -f deploy/docker-compose.yml up --build
```

### Native development

Postgres in Docker; API, worker, and web on the host with hot reload:

```bash
make setup-local
make dev-db
make dev-api      # terminal 1
make dev-worker   # terminal 2
make dev-web      # terminal 3
```

Uses [`deploy/.env.local.example`](./deploy/.env.local.example) (`DATABASE_URL` points at `localhost`).
Vite proxies `/api` and `/auth` to the API on port 8080.

### Storybook (design system)

Component catalog for `apps/web` — run locally while building UI:

```bash
cd apps/web
npm run storybook
```

Open `http://localhost:6006`. Build static catalog for CI: `npm run build-storybook`.

## Services

| Service | Port | Image/build |
|---------|------|-------------|
| postgres | 5432 | postgres:16 |
| api | 8080 | `apps/api` |
| worker | — | `apps/worker` |
| web | 3000 | `apps/web` |

## Environment variables

### Core

| Variable | Description |
|----------|-------------|
| `DATABASE_URL` | Postgres connection string |
| `SESSION_SECRET` | Cookie signing key |
| `WEBHOOK_SECRET` | Alert webhook HMAC/header secret |
| `PUBLIC_URL` | External URL for links in pages |

### OIDC — Google

| Variable | Description |
|----------|-------------|
| `GOOGLE_OIDC_CLIENT_ID` | |
| `GOOGLE_OIDC_CLIENT_SECRET` | |
| `GOOGLE_OIDC_REDIRECT_URL` | `{PUBLIC_URL}/auth/google/callback` |

### OIDC — Slack

| Variable | Description |
|----------|-------------|
| `SLACK_OIDC_CLIENT_ID` | Sign in with Slack |
| `SLACK_OIDC_CLIENT_SECRET` | |
| `SLACK_OIDC_REDIRECT_URL` | |

### OIDC — eXpress

| Variable | Description |
|----------|-------------|
| `EXPRESS_OIDC_ISSUER` | |
| `EXPRESS_OIDC_CLIENT_ID` | |
| `EXPRESS_OIDC_CLIENT_SECRET` | |
| `EXPRESS_OIDC_REDIRECT_URL` | |

### Local dev auth (development only)

| Variable | Description |
|----------|-------------|
| `DEV_AUTH_ENABLED` | Set to `true` to enable **Dev sign in** on `/login` (default off) |
| `DEV_AUTH_DEFAULT_ROLE` | Role for dev login when `?role=` omitted (`admin`, `member`, `viewer`; default `admin`) |
| `DEV_AUTH_EMAIL` | Email stored on the dev user (default `dev@localhost`) |

Requires `PUBLIC_URL` host to be `localhost`, `127.0.0.1`, or `[::1]`. The API logs a warning at startup when enabled. **Never set in production.**

### Integrations (also storable per-row in DB after setup)

- Jira: `JIRA_BASE_URL`, `JIRA_EMAIL`, `JIRA_API_TOKEN`, `JIRA_PROJECT_KEY`
- Slack bot: `SLACK_BOT_TOKEN`, `SLACK_SIGNING_SECRET`
- eXpress bot: `EXPRESS_BOT_ID`, `EXPRESS_BOT_HOST`, `EXPRESS_BOT_SECRET`

## Migrations

Docker Compose runs migrations automatically on `make up` and `make dev-db`.

To apply migrations manually (requires [golang-migrate](https://github.com/golang-migrate/migrate) CLI):

```bash
export DATABASE_URL=postgres://aegis:aegis@localhost:5432/aegis?sslmode=disable
make migrate-up
```

## Local testing without OIDC

To exercise Alerts, Integrations, Dashboard, and Setup without Google/Slack/eXpress app registration:

1. Copy env template: `make setup-local` (or `make setup` for Docker).
2. Set in `.env`:
   ```bash
   DEV_AUTH_ENABLED=true
   PUBLIC_URL=http://localhost:3000   # native dev; use http://localhost:8080 for API-only Docker
   ```
3. Apply migrations (`000010_dev_auth_provider` adds the `dev` user provider):
   ```bash
   make dev-db-down && make dev-db   # reapplies migrations in Docker
   # or: make migrate-up when golang-migrate CLI is installed
   ```
4. Restart the API (`make dev-api` or `make up`).
5. Open http://localhost:3000/login and click **Dev sign in**.
6. Confirm `GET /auth/me` returns your dev user with role `admin`.

If dev login fails, you are redirected to `/login?dev_auth_error=1`.

Integrations and setup test-alert require the admin role; the default dev login uses `admin`. To test
viewer/member RBAC, use `/auth/dev/login?role=viewer` or `?role=member`.

For production-like sign-in testing, configure real OIDC credentials instead (see tables above).

## Setup wizard (Phase 6)

Route: `/setup` in the web app (multi-step wizard; progress in `localStorage`).

1. **Health** — `GET /healthz` on the API (≈ 2 min)
2. **OIDC** — sign in via `/login` and confirm `GET /auth/me` (≈ 30–60 min with IdP app registration, or use **Dev sign in** locally when `DEV_AUTH_ENABLED=true`)
3. **Integrations** — save + test Jira, Slack, eXpress via `/api/v1/integrations` (≈ 2–4 h depending on credentials)
4. **Test alert** — `POST /api/v1/setup/test-alert` from the wizard (≈ 5 min)
5. **Dashboard** — open `/dashboard` and confirm the five widgets load

Target: **≤ 1 working day** for an engineer with IdP and connector credentials ready (NFR-1). Without pre-provisioned credentials, allow half a day for OAuth app setup and token issuance.

## Production notes (MVP)

- TLS termination via reverse proxy (nginx/Caddy) — not bundled
- Back up Postgres volume on schedule
- Rotate API tokens via env reload / redeploy

## References

- Security: [`09-security.md`](./09-security.md)
- Architecture: [`02-architecture.md`](./02-architecture.md)
