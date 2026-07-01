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
| 07 | [`docs/07-setup-deployment.md`](./docs/07-setup-deployment.md) | How IT installs and runs it |
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
- **Auth:** OIDC via Google, Slack, and eXpress.
- **Locales:** English and Russian (`en`, `ru`).
- **Coverage:** `make test` enforces ≥90% unit-test coverage on business logic (NFR-5).
- **Deploy:** Docker Compose — `make setup && make up`, or native dev with `make dev-db` + `make dev-api`.
- **Alert intake:** generic webhook endpoint, compatible with Alertmanager / Grafana / Zabbix payloads.
- **Out:** anything not needed to ship the four MVP features. See `docs/00-product-brief.md` for the
  explicit non-goals.

## Implementation status

**Phases 0–3.5 are implemented** in code. **Phase 4** (alerting workspace) is in progress — see
[`backlog/roadmap.md`](./backlog/roadmap.md).

| Phase | Exit (summary) | Status |
|-------|----------------|--------|
| 0 — Foundation | App runs locally; alert webhook; OIDC API; CI green; Storybook | Done |
| 1 — Shifts & on-call | Teams, rotations, overrides, on-call resolution + calendar UI | Done |
| 2 — Incident spine | Alert → incident → Jira ticket → Slack page → ack + escalation | Done |
| 3 — eXpress connector | Slack + eXpress notifications; test connection per provider | Done |
| 3.5 — Web auth & session | Login page, app shell session, OIDC callback redirect | Done |
| 4 — Alerting workspace | Filters, search, saved views, inline analytics | In progress (AEG-033 done) |
| 5 — L2 ↔ L3 | Handoff, shared timeline, bounce | Not started |
| 6 — Analytics & polish | Dashboard, setup wizard | Not started |

### Backend (`apps/api`, `apps/worker`, `pkg/`)

- **Auth & health:** OIDC login (Google, Slack, eXpress), session cookies, RBAC middleware,
  `/healthz`, `/readyz`, `/metrics`.
- **Alerts:** webhook ingest, list with search/filters/pagination/grouping; enqueues `process_alert` worker job.
- **Shifts:** teams, memberships, schedules, overrides, on-call slots API; `materialise_oncall`
  worker job (on schedule change + nightly).
- **Incidents:** routing rules CRUD; dedup by fingerprint; incident lifecycle (open → acknowledged →
  resolved); timeline events; ack/resolve endpoints.
- **Integrations:** connector registry; Jira ticket provider; Slack + eXpress chat providers;
  interactive ack callbacks; integration CRUD; test connection per provider; eXpress `/link` bootstrap.
- **Worker jobs:** `process_alert`, `escalate_incident`, `materialise_oncall`.

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

### Frontend (`apps/web/`)

- **Design system:** Tailwind tokens, base components, Storybook catalog.
- **Shifts page:** on-call banner + month calendar (rotations and overrides).
- **Incidents page:** status filters, list/detail with timeline, alerts, Jira link, ack/resolve
  actions.
- **Integrations page:** list connectors and test connection (admin); requires sign-in.
- **Web auth:** login page (`/login`), session in app shell, protected `/integrations`, OIDC callback redirect.
- **i18n:** English + Russian locale files for all UI strings.

The web app currently renders **demo fixtures** in `App.tsx` — components are built and tested, but
not yet wired to the API. Backend endpoints are ready for integration.

### Not yet built

- Alerting workspace: advanced filters, saved views, CSV export UI (Phase 4)
- L2 ↔ L3 handoff and bounce (Phase 5)
- Analytics dashboard and setup wizard (Phase 6)
- Web ↔ API client layer for shifts/incidents (demo fixtures today)

## Run locally

Full deployment notes: [`docs/07-setup-deployment.md`](./docs/07-setup-deployment.md).

**Prerequisites:** Docker 24+ with Compose v2. For native dev (option 2): Go 1.25+, Node 20+, and GNU Make (or use `scripts/dev.ps1` on Windows).

### Option 1 — Full stack in Docker (recommended)

Runs Postgres, migrations, API, worker, and web in containers.

```bash
make setup          # copy deploy/.env.example → .env, install deps
# edit .env — at minimum keep SESSION_SECRET and WEBHOOK_SECRET; add OIDC creds to log in
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

On Windows without Make: `.\scripts\dev.ps1 setup` and `.\scripts\dev.ps1 up` (see `.\scripts\dev.ps1` for all commands).

### Other commands

| Command | Description |
|---------|-------------|
| `make install` | Alias for `make setup` |
| `make ps` | Show running Compose services |
| `make migrate-up` | Apply migrations (requires `migrate` CLI + `DATABASE_URL`) |
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
