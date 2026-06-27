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

- **Stack:** Go 1.22+ (API + worker), PostgreSQL 16, React + TypeScript (Vite), Storybook. No Redis.
- **Auth:** OIDC via Google, Slack, and eXpress.
- **Locales:** English and Russian (`en`, `ru`).
- **Coverage:** `make test` enforces ≥90% unit-test coverage on business logic (NFR-5).
- **Deploy:** Docker Compose for the MVP; one `.env`, one `docker compose up`.
- **Alert intake:** generic webhook endpoint, compatible with Alertmanager / Grafana / Zabbix payloads.
- **Out:** anything not needed to ship the four MVP features. See `docs/00-product-brief.md` for the
  explicit non-goals.

## Implementation status

**Phases 0–2 are implemented** in code. Phase 3 (eXpress connector) is next — see
[`backlog/roadmap.md`](./backlog/roadmap.md).

| Phase | Exit (summary) | Status |
|-------|----------------|--------|
| 0 — Foundation | App runs locally; alert webhook; OIDC; CI green; Storybook | Done |
| 1 — Shifts & on-call | Teams, rotations, overrides, on-call resolution + calendar UI | Done |
| 2 — Incident spine | Alert → incident → Jira ticket → Slack page → ack + escalation | Done |
| 3 — eXpress connector | Slack + eXpress notifications; test connection per provider | Not started |
| 4 — Alerting workspace | Filters, search, saved views, inline analytics | Not started |
| 5 — L2 ↔ L3 | Handoff, shared timeline, bounce | Not started |
| 6 — Analytics & polish | Dashboard, setup wizard | Not started |

### Backend (`apps/api`, `apps/worker`, `pkg/`)

- **Auth & health:** OIDC login (Google, Slack, eXpress), session cookies, RBAC middleware,
  `/healthz`, `/readyz`, `/metrics`.
- **Alerts:** webhook ingest, list/detail/export; enqueues `process_alert` worker job.
- **Shifts:** teams, memberships, schedules, overrides, on-call slots API; `materialise_oncall`
  worker job (on schedule change + nightly).
- **Incidents:** routing rules CRUD; dedup by fingerprint; incident lifecycle (open → acknowledged →
  resolved); timeline events; ack/resolve endpoints.
- **Integrations:** connector registry; Jira ticket provider; Slack chat provider (Block Kit page +
  interactive ack callback); integration CRUD; Slack test connection.
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

### Frontend (`apps/web/`)

- **Design system:** Tailwind tokens, base components, Storybook catalog.
- **Shifts page:** on-call banner + month calendar (rotations and overrides).
- **Incidents page:** status filters, list/detail with timeline, alerts, Jira link, ack/resolve
  actions.
- **i18n:** English + Russian locale files for all UI strings.

The web app currently renders **demo fixtures** in `App.tsx` — components are built and tested, but
not yet wired to the API. Backend endpoints are ready for integration.

### Not yet built

- eXpress chat provider and `/callbacks/express/bot` (Phase 3 — AEG-019, AEG-020)
- Alerting workspace: advanced filters, saved views, CSV export UI (Phase 4)
- L2 ↔ L3 handoff and bounce (Phase 5)
- Analytics dashboard and setup wizard (Phase 6)
- Web ↔ API client layer (follow-up after Phase 2 UI stories)

## Run locally

Full setup: [`docs/07-setup-deployment.md`](./docs/07-setup-deployment.md).

```bash
cp deploy/.env.example .env
# Set OIDC credentials, WEBHOOK_SECRET, and optional Jira/Slack tokens
docker compose -f deploy/docker-compose.yml up --build
```

| Service | URL |
|---------|-----|
| Web | http://localhost:3000 |
| API | http://localhost:8080 |
| Storybook | `cd apps/web && npm run storybook` → http://localhost:6006 |

Development gate (lint, typecheck, tests + coverage):

```bash
make lint type test
```

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
├── deploy/                # docker-compose, Dockerfiles, env template
├── scripts/               # coverage gate and helpers
└── .github/               # CI, PR template
```
