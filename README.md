# Aegis — On-Call & Incident Management Platform

> Working codename: **Aegis**. Internal platform for shift scheduling, alert routing, incident
> management, and L2↔L3 coordination, with a web UI and integrations into Jira, Slack, and eXpress.

This repository is **documentation-first**. It is written so AI coding agents (and humans) can pick
up work in a tight, repeatable loop without re-deriving context every time. Start here, then go to
the agent loop.

## What we're building (one paragraph)

A small ops team needs to know who is on call right now, get alerted when things break on the
channels they actually use, turn noisy alerts into tracked incidents (with a Jira ticket created
automatically), and hand work cleanly between L2 and L3 support. Aegis does that and gives the IT
department dashboards to analyse what's happening. The product must be easy to **set up**, easy to
**use**, and easy to **analyse** — those three are the north star for every decision.

## Read in this order

| # | Doc | What it answers |
|---|-----|-----------------|
| — | [`CLAUDE.md`](./CLAUDE.md) | How an agent works in this repo (conventions, the loop, guardrails) |
| 00 | [`docs/00-product-brief.md`](./docs/00-product-brief.md) | Vision, users, scope, success metrics |
| 01 | [`docs/01-prd.md`](./docs/01-prd.md) | Detailed MVP requirements per feature |
| 02 | [`docs/02-architecture.md`](./docs/02-architecture.md) | Components, tech stack, request flows |
| 03 | [`docs/03-data-model.md`](./docs/03-data-model.md) | Entities, schema, relationships |
| 04 | [`docs/04-api-spec.md`](./docs/04-api-spec.md) | REST endpoints and contracts |
| 05 | [`docs/integrations/`](./docs/integrations/) | Jira, Slack, eXpress |
| 06 | [`docs/features/`](./docs/features/) | Shifts, incidents, alerting, L2↔L3 |
| 07 | [`docs/07-setup-deployment.md`](./docs/07-setup-deployment.md) | Local run + DevOps production deploy |
| 08 | [`docs/08-analytics.md`](./docs/08-analytics.md) | Metrics, dashboards, exports |
| 09 | [`docs/09-security.md`](./docs/09-security.md) | OIDC auth, RBAC, secrets, audit |
| 10 | [`docs/10-agent-loop.md`](./docs/10-agent-loop.md) | The development loop in detail |
| 11 | [`docs/11-localization.md`](./docs/11-localization.md) | English + Russian i18n rules |
| 12 | [`docs/12-design-system.md`](./docs/12-design-system.md) | UI tokens, components, patterns (see also [`design_system.html`](./docs/design_system.html)) |
| — | [`backlog/roadmap.md`](./backlog/roadmap.md) | Phases and milestones |
| — | [`backlog/epics/`](./backlog/epics/) | Epics → stories → acceptance criteria |
| — | [`docs/overview.html`](./docs/overview.html) | Visual, scannable map of the whole plan |
| — | [`docs/design_system.html`](./docs/design_system.html) | Visual design system canvas |

## Quick facts

- **Stack:** Go 1.25+ (API + worker), PostgreSQL 16, React + TypeScript (Vite), Storybook. No Redis.
- **Auth:** OIDC via Google, Slack, and eXpress; optional **Dev sign in** on localhost (`DEV_AUTH_ENABLED`).
- **Locales:** English and Russian (`en`, `ru`).
- **Coverage:** `make test` enforces ≥90% unit-test coverage on business logic (NFR-5).
- **Deploy:** Docker Compose (MVP). DevOps runbook below; full notes in
  [`docs/07-setup-deployment.md`](./docs/07-setup-deployment.md).
- **Alert intake:** generic webhook endpoint, compatible with Alertmanager / Grafana / Zabbix payloads.
- **Out:** anything not needed to ship the four MVP features. See `docs/00-product-brief.md` for the
  explicit non-goals.

## Deploy (DevOps)

MVP ships as **Docker Compose**: Postgres 16, schema migrations, API, worker, and web. No Redis.
Helm/Kubernetes is out of scope for MVP (see roadmap *Later*).

**Authoritative detail:** [`docs/07-setup-deployment.md`](./docs/07-setup-deployment.md)
(prerequisites, env reference, migrations, production checklist). Env templates:
[`deploy/.env.example`](./deploy/.env.example).

### Production-like rollout

1. **Host requirements:** Docker 24+ with Compose v2; outbound HTTPS for OIDC IdPs and connectors;
   inbound HTTPS for users and alert webhooks (terminate TLS on a reverse proxy — nginx/Caddy —
   not bundled in Compose).
2. **Secrets / config:** copy `deploy/.env.example` → `.env` on the host (or inject the same keys
   from your secret store). Set strong `SESSION_SECRET` and `WEBHOOK_SECRET`. Set `PUBLIC_URL` to
   the public HTTPS origin users hit (e.g. `https://aegis.example.com`). Configure at least one
   OIDC provider (Google, Slack, and/or eXpress) with redirect URLs under `{PUBLIC_URL}/auth/...`.
   Set `ADMIN_EMAILS` (comma-separated) to the operators who should become **admin** on first
   OIDC sign-in — this bootstraps the first admin. See
   [First admin](./docs/07-setup-deployment.md#first-admin-production).
3. **Never enable in production:** `DEV_AUTH_ENABLED`, `SEED_DEV`, or the Compose `--profile dev`
   alert simulator. Do not point production `DATABASE_URL` at a shared/dev database.
4. **Start stack** (from repo root, `.env` present):

   ```bash
   make up-detached    # builds images, applies golang-migrate, starts postgres/api/worker/web
   make ps             # confirm services
   curl -fsS "$PUBLIC_URL/healthz"   # or http://localhost:8080/healthz behind the proxy
   curl -fsS http://localhost:8080/readyz
   ```

   Equivalent without Make:

   ```bash
   cp deploy/.env.example .env   # first time only
   docker compose -f deploy/docker-compose.yml up --build -d
   ```

5. **Health:** use `GET /healthz` (liveness) and `GET /readyz` (Postgres ready) on the API.
   Prometheus scrape: `GET /metrics` on the API.
6. **Day-2 config (in the UI, not only env):** sign in as admin (the first admin comes from
   `ADMIN_EMAILS`, step 2) → `/integrations` (global Jira/Slack/eXpress) → `/workspaces` (projects
   + connector slots) → teams/schedules → send a test alert. Connector credentials may live in env
   initially; prefer DB-backed config via Integrations after go-live. See
   [`docs/integrations/README.md`](./docs/integrations/README.md).
7. **Upgrade / redeploy:** pull the release tag, rebuild/restart Compose (`make up-detached` or
   `docker compose ... up --build -d`). Migrations run automatically via the `migrate` service
   before API/worker start. Prefer rolling only after a Postgres backup.
8. **Backup:** schedule dumps or snapshots of the `pgdata` volume (Compose volume named `pgdata`).
   Restore by stopping the stack, restoring data, starting again; then confirm `/readyz`.
9. **Rollback:** redeploy the previous image tags/commit; if a migration must reverse, use
   `make migrate-down` (one step) only with a restore plan — downs are provided but not all are
   zero-downtime.

| Service | Port (host) | Role |
|---------|-------------|------|
| postgres | 5432 | State (back this volume up) |
| api | 8080 | HTTP API, OIDC, webhooks, `/healthz` `/readyz` `/metrics` |
| worker | — | Jobs: alert processing, escalation, on-call materialisation, handoff notify |
| web | 3000 | SPA; proxies `/api` and `/auth` to the API (see `deploy/nginx.web.conf`) |

Compose file: [`deploy/docker-compose.yml`](./deploy/docker-compose.yml). Images:
`deploy/Dockerfile.{api,worker,web}`.

## Implementation status

**Phases 0–7 are complete** on `main` (MVP plus local dev auth). See [`backlog/roadmap.md`](./backlog/roadmap.md)
for phase goals; further work is listed there under *Later*.

| Phase | Exit (summary) | Status |
|-------|----------------|--------|
| 0 — Foundation | App runs locally; alert webhook; OIDC API; CI green; Storybook | Done |
| 1 — Shifts & on-call | Teams, rotations, overrides, on-call resolution + calendar UI | Done |
| 2 — Incident spine | Alert → incident → Jira ticket → Slack page → ack + escalation | Done |
| 3 — eXpress connector | Slack + eXpress notifications; test connection per provider | Done |
| 3.5 — Web auth & session | Login page, app shell session, OIDC callback redirect | Done |
| 4 — Alerting workspace | Filters, search, saved views, inline analytics, CSV export | Done (PR #11) |
| 5 — L2 ↔ L3 | Handoff, shared timeline, bounce, handoff analytics | Done (PR #12) |
| 6 — Analytics & polish | Dashboard, setup wizard, a11y pass | Done (PR #13) |
| 7 — Local dev auth | Dev sign-in for localhost without OIDC | Done |

### Backend (`apps/api`, `apps/worker`, `pkg/`)

- **Auth & health:** OIDC login (Google, Slack, eXpress), opt-in dev login for localhost,
  session cookies, RBAC middleware, `/healthz`, `/readyz`, `/metrics`.
- **Alerts:** webhook ingest, list with search/filters/pagination/grouping/analytics; saved views CRUD; CSV export; enqueues `process_alert` worker job.
- **Shifts:** teams, memberships, schedules, overrides, on-call slots API; `materialise_oncall`
  worker job (on schedule change + nightly).
- **Incidents:** routing rules CRUD; dedup by fingerprint; incident lifecycle (open → acknowledged →
  resolved); timeline events; ack/resolve endpoints.
- **Integrations:** connector registry; Jira ticket provider; Slack + eXpress chat providers;
  interactive ack callbacks; integration CRUD; test connection per provider; eXpress `/link` bootstrap.
- **L2 ↔ L3:** handoff and bounce APIs; shared incident timeline; `notify_handoff` worker job.
- **Analytics:** MTTA/MTTR, noise, on-call load, handoffs, overview aggregation; setup test-alert endpoint.
- **Worker jobs:** `process_alert`, `escalate_incident`, `materialise_oncall`, `notify_handoff`.

API contracts: [`docs/04-api-spec.md`](./docs/04-api-spec.md). Env vars: [`deploy/.env.example`](./deploy/.env.example).

### Database (`db/migrations/`)

| Migration | Schema |
|-----------|--------|
| `000001_initial` | users, sessions, jobs, alerts |
| `000002_teams` | teams, team_members |
| `000003_schedules` | schedules, on_call_slots |
| `000004_overrides_oncall` | schedule_overrides |
| `000005_incidents` | incidents, routing_rules, integrations, timeline, incident_alerts |
| `000006_express_link_codes` | express_link_codes for `/link` bootstrap |
| `000007_alert_search_indexes` | alert search backfill + `received_at` index |
| `000008_saved_views` | saved_views for alert workspace |
| `000009_handoffs` | handoffs for L2→L3 tracking and analytics |
| `000010_dev_auth_provider` | allow `dev` OIDC provider for local dev login |
| `000011_user_identities` | `user_identities`, `avatar_url`, `audit_log`; backfill SSO identities |
| `000012_workspaces` | workspaces + team membership |
| `000013_support_tier` | support tier on teams |
| `000014_escalation_paths` | escalation paths |
| `000015_workspace_integrations` | per-workspace integration rows |
| `000016_integration_slot_mode` | slot `mode` (`inherit`/`custom`) + backfill three slots |

### Frontend (`apps/web/`)

- **Design system:** Tailwind tokens, base components, Storybook catalog.
- **Shifts page:** on-call banner + month calendar (rotations and overrides).
- **Incidents page:** status filters, list/detail with timeline, alerts, Jira link, ack/resolve,
  handoff and bounce (demo state in `App.tsx` today).
- **Integrations page:** list connectors and test connection (admin); API-backed; requires sign-in.
- **Alerts page:** filter bar, paginated table, group-by, inline analytics, saved views, CSV export;
  API-backed; requires sign-in.
- **Dashboard page:** five north-star widgets (MTTA, MTTR, noise, on-call load, handoffs,
  escalation); compare-to-previous; drill-down links; API-backed.
- **Setup wizard:** multi-step guided setup (health, auth, integrations, test alert); progress in
  localStorage; API-backed.
- **Web auth:** login page (`/login`) with OIDC providers and **Dev sign in** when enabled,
  session in app shell, protected routes, OIDC callback redirect.
- **i18n:** English + Russian locale files for all UI strings.

**Shifts** and **incidents** pages still use **demo fixtures** in `App.tsx` — UI and handlers are
built and tested; backend endpoints exist for a future API wiring pass.

### Not yet built (post-MVP)

See [`backlog/roadmap.md`](./backlog/roadmap.md) *Later*: Mattermost/Telegram, mobile push,
phone/SMS paging, status pages, runbook automation, multi-tenant SaaS, self-hosted IdP, Helm/K8s
deploy, and related items.

## Run locally

Production/K8s: pull `ghcr.io/btb-hub/aegis` — see
[`docs/07-setup-deployment.md`](./docs/07-setup-deployment.md#production-image-ghcr).

Full deployment notes: [`docs/07-setup-deployment.md`](./docs/07-setup-deployment.md).

**Prerequisites:** Docker 24+ with Compose v2. For native dev (option 2): Go 1.25+, Node 20+, and GNU Make (or use `scripts/dev.ps1` on Windows).

### Option 1 — Full stack in Docker (recommended)

Runs Postgres, migrations, API, worker, and web in containers.

```bash
make setup          # copy deploy/.env.example → .env, install deps
# edit .env — SESSION_SECRET, WEBHOOK_SECRET; OIDC creds for production-like sign-in
# optional: DEV_AUTH_ENABLED=true and PUBLIC_URL=http://localhost:3000 for Dev sign in
make up             # build and start all services (foreground)
# or: make up-detached && make logs
```

| Service | URL |
|---------|-----|
| Web | http://localhost:3000 |
| API | http://localhost:8080 |
| Health | http://localhost:8080/healthz |

Stop: `make down`.

### Option 2 — Native apps + Postgres in Docker

Hot reload for Go and Vite while Postgres runs in a container.

```bash
make setup-local    # copy deploy/.env.local.example → .env (localhost DATABASE_URL)
make dev-db         # Postgres + migrations in Docker (port 5432)

# three terminals:
make dev-api        # API on :8080
make dev-worker     # background job worker
make dev-web        # Vite dev server on :3000 (proxies /api and /auth to :8080)
```

Stop Postgres: `make dev-db-down`.

#### Sign in without OIDC (local dev)

Use this to open Alerts, Integrations, Dashboard, and Setup without Google/Slack/eXpress app registration.

1. In `.env` (from `make setup-local`):
   ```bash
   DEV_AUTH_ENABLED=true
   PUBLIC_URL=http://localhost:3000
   ```
2. Ensure migrations are applied (`make dev-db` runs them automatically; includes `000010_dev_auth_provider`).
3. Restart the API after changing `.env`.
4. Open http://localhost:3000/login and click **Dev sign in**.

You get an admin session on localhost only. Do not enable `DEV_AUTH_ENABLED` in production. Full notes:
[`docs/07-setup-deployment.md`](./docs/07-setup-deployment.md#local-testing-without-oidc).

#### Seed dev users (local directory)

After migrations, populate SSO-like users for team pickers and shifts testing without real OIDC sign-ins:

```bash
make seed-dev
```

Requires `DATABASE_URL` and `PUBLIC_URL` pointing at localhost (same guard as dev auth), or set
`SEED_DEV=1` to override the host check. Idempotent — safe to re-run. Seeds four users: Google,
Slack (with `slack_user_id` + avatar), eXpress (with `express_user_huid` + avatar), and a dev
admin row (`dev@localhost`).

**Typical local on-call flow:** `make seed-dev` → sign in (dev or OIDC) → **Setup** wizard or **Teams**
→ create a team and members → open **Shifts** → create a weekly schedule and optional overrides.
The calendar reads `GET /teams/{id}/on-call/*`; no demo fixtures on the production route.

On Windows without Make: `.\scripts\dev.ps1 setup` and `.\scripts\dev.ps1 up` (see `.\scripts\dev.ps1` for all commands).

### Other commands

| Command | Description |
|---------|-------------|
| `make install` | Alias for `make setup` |
| `make ps` | Show running Compose services |
| `make migrate-up` | Apply migrations (requires `migrate` CLI + `DATABASE_URL`) |
| `make seed-dev` | Upsert local dev user directory (localhost guard; see below) |
| `make lint type test` | CI gate — lint, typecheck, tests + coverage |

Storybook: `cd apps/web && npm run storybook` → http://localhost:6006

### Compose files

| File | Purpose |
|------|---------|
| [`deploy/docker-compose.yml`](./deploy/docker-compose.yml) | Full stack (`make up`) |
| [`deploy/docker-compose.dev.yml`](./deploy/docker-compose.dev.yml) | Postgres only (`make dev-db`) |
| [`deploy/.env.example`](./deploy/.env.example) | Env template for Docker (`DATABASE_URL` host `postgres`) |
| [`deploy/.env.local.example`](./deploy/.env.local.example) | Env template for native dev (`DATABASE_URL` host `localhost`) |

## Repo layout

```
aegis/
├── docs/                  # the spec (source of truth)
├── backlog/               # roadmap + epics/stories the agents pull from
├── pkg/                   # shared Go packages (config, db, integrations, routing, oncall, i18n)
├── apps/
│   ├── api/               # Go/Gin HTTP service
│   ├── worker/            # Go job poller (alert processing, escalations, on-call materialisation)
│   └── web/               # React frontend + Storybook
├── db/                    # golang-migrate SQL migrations
├── deploy/                # docker-compose, Dockerfiles, env templates
├── scripts/               # coverage gate, env helpers for local dev
└── .github/               # CI, PR template
```
