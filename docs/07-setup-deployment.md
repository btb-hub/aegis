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

### Dev user seeds (development only)

| Variable | Description |
|----------|-------------|
| `SEED_DEV` | Set to `1` to allow `make seed-dev` when `PUBLIC_URL` is not localhost (still required) |

See [Dev user seeds](#dev-user-seeds-local-only) below.

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

## Dev user seeds (local only)

Populate a fixed directory of users for team member pickers and on-call setup without OIDC sign-ins:

```bash
make seed-dev
# or: go run ./apps/api/cmd/seed-dev   (with .env loaded)
```

**Guard:** runs only when `PUBLIC_URL` host is `localhost`, `127.0.0.1`, or `[::1]`, unless
`SEED_DEV=1` is set (still requires `PUBLIC_URL`). **Never run against production.**

**Seeded users (idempotent):**

| Display name | Email | Provider | Notes |
|--------------|-------|----------|-------|
| Alice Google | `alice@seed.local` | google | avatar URL |
| Bob Slack | `bob@seed.local` | slack | `slack_user_id`, avatar |
| Carol eXpress | `carol@seed.local` | express | `express_user_huid`, avatar, locale `ru` |
| Local Admin | `dev@localhost` | dev | admin role for RBAC tests |

Re-run after schema changes or to reset profile fields to the canonical seed values.

## Alert simulator (dev only)

The alert simulator lives under `devtools/alert-simulator/` — it is **not** shipped in production
binaries. It integrates with Aegis through the public **alert webhook** and **HTTP API** (dev auth
for bootstrap).

```bash
# Terminal 1–2: api + worker (see Native dev above)
make dev-api
make dev-worker

# Terminal 3: ensure routing via API (requires DEV_AUTH_ENABLED=true)
make dev-simulator-bootstrap

# Send one random alert
make simulate-alert

# Or run continuously (default every 30s)
make dev-simulator
```

**Prerequisites:** api and worker running; `WEBHOOK_SECRET` in `.env`; routing rules for simulator
team labels. Run `make dev-simulator-bootstrap` once on a fresh setup — it creates NOC, Helpdesk,
Ops, and Platform teams with routing rules (`team=noc`, `l1`, `ops`, `platform`) and escalation
paths (NOC→Helpdesk→Ops→Platform) through the HTTP API.

| Variable | Purpose |
|----------|---------|
| `WEBHOOK_SECRET` | Required — same as API |
| `AEGIS_API_URL` | API base for bootstrap (default `PUBLIC_URL` or `http://localhost:8080`) |
| `AEGIS_WEBHOOK_URL` | Full webhook URL (default `{API}/api/v1/alerts/webhook`) |
| `ALERT_SIM_INTERVAL` | Loop interval when running without flags (default `30s`) |
| `ALERT_SIM_TEAM` / `ALERT_SIM_PROJECT` | Override labels for all scenarios (default: per-scenario tier routing) |

**CLI flags:**

```bash
go run ./devtools/alert-simulator/cmd/alert-simulator -list
go run ./devtools/alert-simulator/cmd/alert-simulator -bootstrap-only
go run ./devtools/alert-simulator/cmd/alert-simulator -scenario high_cpu
go run ./devtools/alert-simulator/cmd/alert-simulator -once
go run ./devtools/alert-simulator/cmd/alert-simulator -interval 1m
```

**Docker (optional dev profile):**

```bash
make up-dev              # full stack + alert simulator (foreground)
make up-dev-detached     # same, detached
```

Or:

```bash
docker compose -f deploy/docker-compose.yml --profile dev up --build
```

Built-in scenarios include high CPU, disk full, OOM kills, HTTP 5xx spikes, DB connection pool
exhaustion, certificate expiry, queue backlog, replication lag, and DNS failures.

## Configuration model (preferred)

**Configure each area on its own admin page.** Do not require the multi-step setup wizard to change
one setting (for example Jira credentials or workspace membership).

| Area | Preferred UI | API |
|------|--------------|-----|
| Connectors (Jira, Slack, eXpress) | `/integrations` — create, edit, test, enable/disable | `/api/v1/integrations` |
| Teams / shifts / users | Dedicated pages under the app shell | respective `/api/v1/...` routes |
| Workspaces / escalation / routing | **`/workspaces`** (create, bind teams) + team path admin + workspace routing | workspace / path / routing APIs |

The setup wizard (`/setup`) is an **optional first-run checklist**. It must not gate connector or
other day-2 configuration. Product decisions:
[`EPIC-12`](../backlog/epics/EPIC-12-workspace-admin.md) (workspaces),
[`EPIC-13`](../backlog/epics/EPIC-13-integration-admin.md) (integrations).

**Workspaces:** Primary path is **Workspaces** (`/workspaces`) — create a project and assign existing
teams. The wizard keeps a **Quick setup** shortcut (workspace + L2/L3 + escalation path in one click)
for greenfield installs. If a non-default workspace already has teams, the wizard step can be skipped.

### Suggested independent first-day path (NFR-1)

1. **Health** — `GET /healthz` (≈ 2 min)
2. **Sign-in** — `/login` (OIDC, or **Dev sign in** when `DEV_AUTH_ENABLED=true`) and confirm `GET /auth/me` (≈ 30–60 min with IdP, or seconds with dev auth)
3. **Workspace & teams** — open **`/workspaces`** (or wizard quick L2/L3 setup) (≈ 15–30 min)
4. **Integrations** — open **`/integrations`**, save + test Jira / Slack / eXpress (≈ 2–4 h depending on credentials). Do **not** require `/setup`.
5. **Test alert** — `POST /api/v1/setup/test-alert` (admin; also reachable from the wizard when used) (≈ 5 min)
6. **Dashboard** — open `/dashboard` and confirm the five widgets load

Target: **≤ 1 working day** for an engineer with IdP and connector credentials ready (NFR-1). Without pre-provisioned credentials, allow half a day for OAuth app setup and token issuance.

## Setup wizard (optional)

Route: `/setup` (multi-step; progress in `localStorage`). Useful as a guided tour for a greenfield
install. Steps still cover health → OIDC → workspace/teams → integrations → test alert → dashboard, but
**each capability above is also available without the wizard**. Prefer deep links from wizard steps into
the dedicated admin pages (`/workspaces`, `/integrations` as those pages mature).

## Production deploy (DevOps)

MVP target is a single-host **Docker Compose** stack. There is no Helm chart yet. A short checklist
also lives in the root [`README.md`](../README.md#deploy-devops).

### Topology

| Unit | Image / build | Notes |
|------|---------------|--------|
| `postgres` | `postgres:16` | Named volume `pgdata`; change default password before internet exposure |
| `migrate` | `migrate/migrate:v4.17.1` | Runs `up` once per `compose up`; exits 0 before API/worker start |
| `api` | `deploy/Dockerfile.api` | Port 8080; loads root `.env` |
| `worker` | `deploy/Dockerfile.worker` | Same env + DB as API; no public port |
| `web` | `deploy/Dockerfile.web` | Port 3000; nginx proxies `/api` and `/auth` to `api:8080` |

Do **not** start the `alert-simulator` service in production (`--profile dev` only).

### Pre-flight

1. Provision TLS at a reverse proxy (nginx, Caddy, cloud LB). Point DNS at the proxy; set
   `PUBLIC_URL` to that HTTPS origin.
2. Create `.env` from [`deploy/.env.example`](../deploy/.env.example) (or equivalent secrets in
   your platform). Required for a usable install:
   - `SESSION_SECRET`, `WEBHOOK_SECRET` — long random values
   - `PUBLIC_URL` — public origin (no trailing path)
   - At least one OIDC client set (`GOOGLE_*`, `SLACK_*`, and/or `EXPRESS_*`) with redirect URLs
     matching `{PUBLIC_URL}/auth/<provider>/callback`
3. Confirm firewall: 443 (users + IdP callbacks + optional webhook), and optionally restrict
   direct 5432/8080/3000 to the proxy/admin network only.
4. Confirm `DEV_AUTH_ENABLED=false` (or unset) and that `SEED_DEV` is not set.

Compose still injects `DATABASE_URL=postgres://aegis:aegis@postgres:5432/aegis?sslmode=disable`
for in-network app containers. Change the Postgres password via image env + matching URL if the
DB port is reachable outside the Compose network.

### Deploy / upgrade

```bash
# On the target host, repo checked out at the release tag
cp deploy/.env.example .env   # first install only; then edit secrets
# edit .env

make up-detached              # build, migrate, start detached
# or: docker compose -f deploy/docker-compose.yml up --build -d

make ps
curl -fsS https://<public-host>/healthz   # via proxy, or :8080 on the host
curl -fsS http://127.0.0.1:8080/readyz
```

**Upgrade procedure:** take a Postgres backup → `git fetch && git checkout <tag>` →
`make up-detached` (rebuilds images and runs pending migrations) → check `/readyz` and a signed-in
smoke test (login, open dashboard, send webhook with `WEBHOOK_SECRET`).

**Manual migration** (emergency / non-Compose):

```bash
export DATABASE_URL=postgres://aegis:<password>@<host>:5432/aegis?sslmode=require
make migrate-up               # requires golang-migrate CLI
# one-step reverse if needed (plan restore first):
make migrate-down
```

### Operations checklist

| Concern | Guidance |
|---------|----------|
| Liveness | `GET /healthz` on API |
| Readiness | `GET /readyz` (fails if DB unreachable) |
| Metrics | `GET /metrics` (Prometheus text) |
| Logs | `make logs` or `docker compose -f deploy/docker-compose.yml logs -f api worker` |
| Backups | Snapshot / `pg_dump` the `pgdata` volume on a schedule; test restore |
| Secret rotation | Update `.env` (or secret store), recreate `api`/`worker` containers |
| Connector secrets | Prefer admin UI `/integrations` + workspace slots after go-live; env vars remain optional bootstrap |
| Alert intake | External systems POST to `{PUBLIC_URL}/api/v1/alerts/webhook` with configured secret |
| Hardening | TLS at proxy; do not expose Postgres publicly; disable dev auth and the `dev` Compose profile |

### Rollback

1. Redeploy the previous known-good git tag / images with `make up-detached`.
2. If a new migration already applied and the old code cannot run against the new schema, restore
   Postgres from the pre-upgrade backup, then redeploy the old tag. Use `migrate-down` only when
   you understand the specific down SQL (provided next to each `*.up.sql`).

## References

- Security: [`09-security.md`](./09-security.md)
- Architecture: [`02-architecture.md`](./02-architecture.md)
- Root README DevOps section: [`../README.md`](../README.md#deploy-devops)
