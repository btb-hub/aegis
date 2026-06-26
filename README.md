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

- **Stack:** Go 1.22+ (API + worker), PostgreSQL 16, React + TypeScript (Vite). No Redis.
- **Auth:** OIDC via Google, Slack, and eXpress.
- **Locales:** English and Russian (`en`, `ru`).
- **Coverage:** `make test` enforces ≥90% unit-test coverage on business logic (NFR-5).
- **Deploy:** Docker Compose for the MVP; one `.env`, one `docker compose up`.
- **Alert intake:** generic webhook endpoint, compatible with Alertmanager / Grafana / Zabbix payloads.
- **Out:** anything not needed to ship the four MVP features. See `docs/00-product-brief.md` for the
  explicit non-goals.

## Repo layout (target)

```
aegis/
├── docs/                  # the spec (this is the source of truth)
├── backlog/               # roadmap + epics/stories the agents pull from
├── apps/
│   ├── api/               # Go/Gin HTTP service
│   ├── worker/            # Go job poller (notifications, escalations, sync)
│   └── web/               # React frontend
├── db/                    # migrations + sqlc queries
├── deploy/                # docker-compose, env templates
└── .github/               # CI, PR template
```

The `apps/`, `db/`, and `deploy/` trees implement Phase 0 foundation. The docs
and backlog drive what ships next.
