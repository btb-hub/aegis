# Aegis — On-Call & Incident Management Platform

> Working codename: **Aegis**. Internal platform for shift scheduling, alert routing, incident
> management, and L2↔L3 coordination, with a web UI and integrations into Jira, Slack, and eXpress.

A small ops team needs to know who is on call right now, get alerted when things break on the
channels they actually use, turn noisy alerts into tracked incidents (with a Jira ticket created
automatically), and hand work cleanly between L2 and L3 support. Aegis does that and gives the IT
department dashboards to analyse what's happening. The product must be easy to **set up**, easy to
**use**, and easy to **analyse**.

This repository is **documentation-first**. Specs live in `docs/` and `backlog/`. The
how-tos are the **user manual** for configuring the on-call workflow in the product.

## How-to: set up the workflow

These pages are for admins and on-call engineers using Aegis, not for compiling the repo.
Read them on GitHub (Markdown). HTML copies in the same folder are for a docs host.

| Guide | What it covers |
|-------|----------------|
| [Set up the workflow](./docs/how-to/setup.md) | Admin screens in order: workspace, L2/L3 teams, members, escalation path, weekly rotation, routing rule, test alert |
| [Routing rules](./docs/how-to/routing.md) | Filled-in forms: `team=platform` → Platform L2; NOC-first; shared DevOps |
| [Escalation paths](./docs/how-to/paths.md) | L2 → L3, NOC chain, shared L3 |
| [Work an incident](./docs/how-to/incidents.md) | Acknowledge, hand off, bounce, resolve |
| [Connect Jira, Slack, eXpress](./docs/how-to/connect.md) | Global bots, Jira Server/DC or Cloud, workspace slots, Account paging connections, shared channels |
| [When it does not route](./docs/how-to/troubleshooting.md) | Alerts without incidents, empty on-call banner, missing handoff button |

Hub: [`docs/how-to/README.md`](./docs/how-to/README.md). Russian: [`docs/how-to/ru/README.md`](./docs/how-to/ru/README.md). Installing the software (Docker, env, GHCR): [`docs/07-setup-deployment.md`](./docs/07-setup-deployment.md). Day-to-day by role: [`docs/user-guide.md`](./docs/user-guide.md).

## Set up locally

**Prerequisites:** Docker 24+ with Compose v2. For native hot reload (option 2): Go 1.25+, Node 20+,
and GNU Make (or `.\scripts\dev.ps1` on Windows).

### Option 1 — Full stack in Docker (recommended)

```bash
make setup          # copy deploy/.env.example → .env, install deps
```

Edit `.env`:

```bash
DEV_AUTH_ENABLED=true
```

```bash
make up             # build, migrate, start postgres/api/worker/web
# or: make up-detached && make logs
```

| Service | URL |
|---------|-----|
| Web | http://localhost:3000 |
| API | http://localhost:8080 |
| Health | http://localhost:8080/healthz |
| Ready | http://localhost:8080/readyz |

Stop: `make down`.

Open http://localhost:3000/login and click **Dev sign in**. Never set `DEV_AUTH_ENABLED` in production.

### Option 2 — Native apps + Postgres in Docker

Hot reload for Go and Vite. Postgres stays in a container.

```bash
make setup-local    # copy deploy/.env.local.example → .env (DATABASE_URL host localhost)
make dev-db         # Postgres + migrations (port 5432)

# three terminals:
make dev-api        # API on :8080
make dev-worker     # background job worker
make dev-web        # Vite on :3000 (proxies /api and /auth to :8080)
```

Stop Postgres: `make dev-db-down`. Same Dev sign in as option 1. Configure the app in **Settings**.

<details>
<summary>Seed users, Windows, and extra commands</summary>

<br>

After migrations and a local dev sign-in, populate a user directory for team pickers:

```bash
make seed-dev
```

Guarded to the database's localhost public URL (or `SEED_DEV=1`). Idempotent. Seeds Alice (Google), Bob Slack,
Carol eXpress, and Local Admin (`dev@localhost`).

**Typical local on-call flow:** sign in → `make seed-dev` → follow
[`docs/how-to/setup.md`](./docs/how-to/setup.md) (workspace, teams, shifts, routing, test alert).

On Windows without Make: `.\scripts\dev.ps1 setup` and `.\scripts\dev.ps1 up`.

| Command | Description |
|---------|-------------|
| `make install` | Alias for `make setup` |
| `make ps` / `make logs` | Compose status / follow logs |
| `make migrate-up` | Apply migrations (`migrate` CLI + `DATABASE_URL`) |
| `make lint type test` | CI gate — lint, typecheck, tests + coverage |
| `make up-dev` | Full stack + alert simulator (dev profile, not production) |

Storybook: `cd apps/web && npm run storybook` → http://localhost:6006

| File | Purpose |
|-------|---------|
| [`deploy/docker-compose.yml`](./deploy/docker-compose.yml) | Full stack (`make up`) |
| [`deploy/docker-compose.dev.yml`](./deploy/docker-compose.dev.yml) | Postgres only (`make dev-db`) |
| [`deploy/.env.example`](./deploy/.env.example) | Docker (`DATABASE_URL` host `postgres`) |
| [`deploy/.env.local.example`](./deploy/.env.local.example) | Native (`DATABASE_URL` host `localhost`) |

</details>

## Deploy (DevOps)

Use **Docker Compose** for local source builds. Production uses the GHCR all-in-one image and an
**external** Postgres. Authoritative detail: [`docs/07-setup-deployment.md`](./docs/07-setup-deployment.md).

1. For an existing installation, follow the [manual configuration migration](./docs/configuration-migration.md)
   before starting updated services: backup → schema migrations → dry-run against the actual
   production `.env` → apply → deploy → verify → remove migrated env variables.
   For a fresh installation, use external PostgreSQL and `AEGIS_BOOTSTRAP_TOKEN` generated from
   at least 32 random bytes. Open `/bootstrap`, submit the token through the form, configure
   a provider, and verify sign-in with the first administrator's email. Follow the
   [production image instructions](./docs/07-setup-deployment.md#production-image-ghcr) or
   [Kubernetes example](./deploy/k8s/aegis.yaml).
2. Run **one replica**. The image applies migrations at startup, then runs API, worker, and web.
   Container port **3000** serves the UI, `/healthz`, `/readyz`, and `/metrics`.
   Back up external PostgreSQL before upgrades; migration `000022` adds database settings,
   provider drafts, authorization attempts, and permanent installation state.
3. Keep `DEV_AUTH_ENABLED`, `SEED_DEV`, and the alert simulator disabled. Allow provider HTTPS
   traffic and signed bot callbacks through the ingress. See the deployment guide for proxy rules.
4. Sign in as admin, configure bots under **Settings → Integrations**, then workspaces, teams, and schedules.
   Each responder connects **Account → Paging messengers**. Verify a routed test alert, Jira
   issue, personal page, shared channel post, and private acknowledgement result.

Shared messages and chat acknowledgements require the worker. See
[`docs/features/incident-chat-notifications.md`](./docs/features/incident-chat-notifications.md)
for delivery retries and [`docs/13-on-call-runbook.md`](./docs/13-on-call-runbook.md) for operations.

Application configuration lives in PostgreSQL and is edited under **Settings / Настройки**.
Provider edits require a successful **Test sign-in** before activation. Blank secret inputs
preserve stored values. Behavior changes apply to subsequent requests and jobs; listener
changes require an API restart. `/setup` redirects to Settings and `/integrations` redirects
to its Integrations section. The setup wizard has been removed. See the
[settings feature guide](./docs/features/settings.md) and [migration manual](./docs/configuration-migration.md).

## Quick facts

- **Stack:** Go 1.25+ (API + worker), PostgreSQL 16, React + TypeScript (Vite), Storybook. No Redis.
- **Auth:** OIDC via Google, Slack, and eXpress; optional **Dev sign in** on localhost (`DEV_AUTH_ENABLED`).
- **Locales:** English and Russian (`en`, `ru`).
- **Coverage:** `make test` enforces ≥90% unit-test coverage on business logic (NFR-5).
- **Alert intake:** generic webhook, compatible with Alertmanager / Grafana / Zabbix payloads.

## Read in this order

| # | Doc | What it answers |
|---|-----|-----------------|
| — | [`docs/how-to/README.md`](./docs/how-to/README.md) | User manual (EN + RU): workflow, routing rules, escalation paths |
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
| 12 | [`docs/12-design-system.md`](./docs/12-design-system.md) | UI tokens, components, patterns |
| — | [`docs/user-guide.md`](./docs/user-guide.md) | Day-to-day use by role |
| 13 | [`docs/13-on-call-runbook.md`](./docs/13-on-call-runbook.md) | Действия дежурного инженера и первичная диагностика Aegis |
| — | [`backlog/roadmap.md`](./backlog/roadmap.md) | Phases and milestones |
| — | [`backlog/epics/`](./backlog/epics/) | Epics → stories → acceptance criteria |
| — | [`docs/overview.md`](./docs/overview.md) | Scannable map of the whole plan |

## Implementation status

**Phases 0–7 are complete** on `main` (MVP plus local dev auth). See [`backlog/roadmap.md`](./backlog/roadmap.md)
for phase goals; further work is listed there under *Later*.

Subsequent releases add workspace administration, API-backed incident and shift screens,
Account paging connections, and shared incident notifications. The capability lists below
describe the current implementation, beyond the original MVP phase table.

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
- **Integrations:** Jira Server/DC (REST v2) and Cloud (REST v3); Slack + eXpress bots;
  global and workspace Inherit/Custom settings; admin connection tests; Slack callback verification
  with the incident workspace's saved signing secret.
- **Paging connections:** Slack / eXpress browser authorization from Account; connect, replace,
  disconnect; independent from sign-in providers. Legacy eXpress `/link` remains API-compatible.
- **Shared notifications:** opening, acknowledgement, resolution, and escalation posts to the
  team's Slack channel and global eXpress on-call group; durable delivery and chat acknowledgement
  queues, bounded retries, and private result feedback.
- **L2 ↔ L3:** handoff and bounce APIs; shared incident timeline; `notify_handoff` worker job.
- **Analytics:** MTTA/MTTR, noise, on-call load, handoffs, overview aggregation; setup test-alert endpoint.
- **Worker jobs:** `process_alert`, `escalate_incident`, `materialise_oncall`, `notify_handoff`,
  `notify_incident`, `publish_oncall`.
  Independent loops process shared incident messages, chat acknowledgements, and feedback.
  On-call announcements use each team's Slack channel/user group and the global eXpress group.

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
| `000017_routing_rules_cross_workspace` | workspace-owned routing rules and explicit cross-workspace targets |
| `000018_team_chat_channels` | team chat destinations and last announced on-call users |
| `000019_slack_team_tag` | Slack user group for team mentions |
| `000020_paging_connections` | session-bound paging authorizations and durable Slack connection management |
| `000021_incident_chat_outboxes` | shared incident events, per-destination deliveries, queued chat acknowledgements and feedback |

### Frontend (`apps/web/`)

- **Design system:** Tailwind tokens, base components, Storybook catalog.
- **Shifts:** team landing page, on-call banner, month calendar, rotations and overrides;
  API-backed team calendars with shared design-system components.
- **Incidents page:** status filters, list/detail with timeline, alerts, Jira link, ack/resolve,
  handoff and bounce; API-backed, responsive detail view; message links select the incident,
  including resolved incidents.
- **Administration:** workspaces, teams, members, support tiers, escalation paths, routing rules,
  and user roles; protected screens with admin-only configuration controls.
- **Account:** profile, language, connected sign-in providers, and Slack / eXpress paging
  connections with Connect, Replace, and Disconnect actions.
- **Integrations page:** list connectors and test connection (admin); API-backed; requires sign-in.
- **Alerts page:** filter bar, paginated table, group-by, inline analytics, saved views, CSV export;
  API-backed; requires sign-in.
- **Dashboard page:** five north-star widgets (MTTA, MTTR, noise, on-call load, handoffs,
  escalation); compare-to-previous; drill-down links; API-backed.
- **Settings:** admin sign-in drafts and tests, access rules, integrations, behavior, deployment
  settings and diagnostics; PostgreSQL-backed with a manual production env importer.
- **Web auth:** login page (`/login`) with OIDC providers and **Dev sign in** when enabled,
  session in app shell, protected routes, OIDC callback redirect.
- **i18n:** English + Russian locale files for all UI strings.

### Not yet built (post-MVP)

See [`backlog/roadmap.md`](./backlog/roadmap.md) *Later*: Mattermost/Telegram, mobile push,
phone/SMS paging, status pages, runbook automation, multi-tenant SaaS, self-hosted IdP, Helm charts,
and related items.

## Repo layout

```
aegis/
├── docs/                  # spec (source of truth) + how-to (Markdown + HTML)
├── backlog/               # roadmap + epics/stories the agents pull from
├── pkg/                   # shared Go packages (config, db, integrations, routing, oncall, i18n)
├── apps/
│   ├── api/               # Go/Gin HTTP service
│   ├── worker/            # Go jobs + shared incident delivery, chat ack and feedback loops
│   └── web/               # React frontend + Storybook
├── db/                    # golang-migrate SQL migrations
├── deploy/                # docker-compose, Dockerfiles, env templates
├── scripts/               # coverage gate, env helpers for local dev
└── .github/               # CI, PR template
```
