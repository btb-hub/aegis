# Product brief

## Vision

Aegis is an internal on-call and incident management platform for a small IT operations team. It
answers three questions in one place: who is on call right now, what is broken, and who owns fixing
it — with a full audit trail from alert to resolution.

## Users

| Persona | Needs |
|---------|-------|
| **IT admin** | Set up teams, rotations, integrations, and routing in under one working day. |
| **On-call engineer (L2/L3)** | Get paged on Slack or eXpress, acknowledge, work the incident, hand off cleanly. |
| **IT manager** | See MTTA/MTTR, noise, load fairness, L2→L3 handoffs, escalation patterns. |

## North stars

1. **Easy to set up** — one `.env`, one `docker compose up`, guided wizard, first routed alert within one working day.
2. **Easy to use** — one screen per job; acknowledge from chat; L2 hands to L3 in one click.
3. **Easy to analyse** — five dashboard questions answered without a spreadsheet.

## Languages

The web UI and chat page templates are available in **English** and **Russian**. Users pick a language
in the app; the choice is saved to their profile. See [`11-localization.md`](./11-localization.md).

## MVP scope (four features)

1. **Shifts calendar** — rotations, overrides, DST-correct "who's on call now".
2. **Incident management** — alert → dedup → incident → Jira ticket → page → ack → escalate.
3. **Alerting workspace** — filter, search, group, saved views, inline analytics, CSV export.
4. **L2↔L3 transparency** — one-action handoff, shared timeline, handoff analytics.

## Integrations (MVP)

| Type | Provider |
|------|----------|
| Ticket | Jira |
| Chat | Slack, eXpress |

Sign-in uses OIDC via **Google**, **Slack**, and **eXpress** (same three providers; chat credentials are separate from OIDC client config).

## Success metrics

| Metric | Target |
|--------|--------|
| Setup time (clean host → first routed alert) | ≤ 1 working day (NFR-1) |
| MTTA / MTTR | Visible on dashboard; trend vs previous period |
| L2→L3 time-to-first-response | Per-handoff analytics |
| Alert list p95 (10k rows) | < 500 ms (NFR-2) |

## Non-goals (MVP)

- Mattermost, Telegram, or other chat connectors
- Local username/password auth or self-hosted IdP (Keycloak, LDAP, Authentik)
- Redis or other auxiliary datastores
- Native mobile push, SMS/phone paging
- Multi-tenant SaaS, status pages, runbook automation
- Helm charts and multi-replica Kubernetes operators (single all-in-one image + sketch manifests are in-scope; Compose remains the local source-build path)
- Locales beyond English and Russian

## References

- Requirements: [`01-prd.md`](./01-prd.md)
- Localization: [`11-localization.md`](./11-localization.md)
- Design system: [`12-design-system.md`](./12-design-system.md), [`design_system.html`](./design_system.html)
- Roadmap: [`../backlog/roadmap.md`](../backlog/roadmap.md)
