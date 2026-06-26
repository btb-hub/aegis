# Roadmap

Ordered so each phase ends with something demonstrable. The agent loop pulls stories top-down; finish
a phase's "exit" before moving on. Epics and their stories are in [`epics/`](./epics/).

## Guiding order

Build a **thin vertical slice first** (one alert can become one incident with one connector and you
can see who's on call), then thicken it. Don't build all the models, then all the APIs — build
features end to end.

## Phase 0 — Foundation

*Exit: the app runs locally, an alert can be POSTed and stored, OIDC auth works, CI is green, Storybook documents the design system.*

- Repo scaffolding (`apps/api`, `apps/worker`, `apps/web`, `deploy/`, `db/`), Docker Compose dev stack.
- Config, Postgres + golang-migrate baseline, Postgres `jobs` table for async work.
- OIDC auth (Google, Slack, eXpress) + session + RBAC skeleton, `/healthz` `/readyz` `/metrics`.
- Web i18n scaffold: English + Russian (`react-i18next`, `pkg/i18n` for chat templates).
- Design system scaffold: Tailwind tokens + base UI components per `docs/12-design-system.md`.
- **Storybook** for base components (Button, Input, SeverityTag, Toast, shell) — required before Phase 1.
- CI: lint, type, test gate. PR template. `make` targets.
- Generic alert webhook that validates + stores a raw alert (no processing yet).

→ Epic: [EPIC-01 Foundation](./epics/EPIC-01-foundation.md)

## Phase 1 — Shifts & on-call

*Exit: admins define teams + a rotation; "who's on call now" is correct, including DST + an override.*

**Gate:** Phase 0 Storybook story ([AEG-056](./epics/EPIC-01-foundation.md)) must be `Done` before picking EPIC-02 stories.

- Teams, memberships, schedules, overrides, on-call resolution + materialisation.
- Calendar API/view and "who's on call now" view.

→ Epic: [EPIC-02 Shifts](./epics/EPIC-02-shifts.md)

## Phase 2 — Alert → incident → first connector → Jira (the spine)

*Exit: a firing alert creates one incident, makes a Jira ticket, and pages the on-call on Slack; ack works.*

- Alert processing: dedup/group, routing rules, incident creation + lifecycle, timeline.
- Jira ticket provider (create + link).
- Slack chat connector end to end with Acknowledge.
- Escalation timer + one escalation step.

→ Epics: [EPIC-04 Incidents](./epics/EPIC-04-incidents.md), [EPIC-03 Integrations](./epics/EPIC-03-integrations.md)

## Phase 3 — eXpress connector

*Exit: notifications fan out to Slack and eXpress; each has Test connection + Acknowledge.*

- eXpress via BotX HTTP API (HMAC token, bot endpoints, bubbles, `/link` bootstrap).
- Per-connector Test connection; graceful degradation + retry/backoff.

→ Epic: [EPIC-03 Integrations](./epics/EPIC-03-integrations.md)

## Phase 4 — Alerting workspace

*Exit: filter, search, group, saved views, inline analytics, CSV export — fast over large volumes.*

- Indexed list (labels GIN, search tsv), filters, grouping, saved views, inline analytics + export.

→ Epic: [EPIC-05 Alerting](./epics/EPIC-05-alerting.md)

## Phase 5 — L2 ↔ L3 transparency

*Exit: L2 hands an incident to the right L3 on-call in one action; both see one shared timeline;
handoff analytics work.*

- Handoff action + record, shared incident view, reassign/bounce, handoff analytics.

→ Epic: [EPIC-06 L2↔L3](./epics/EPIC-06-l2-l3.md)

## Phase 6 — Analytics & polish

*Exit: the five north-star questions answered on one dashboard; setup wizard end to end; docs match.*

- Overview dashboard (MTTA/MTTR/escalation/noise/load), drill-down, compare-to-previous.
- Setup wizard incl. "Send test alert"; accessibility pass; setup-time check against NFR-1.

→ Epic: [EPIC-07 Analytics & Setup](./epics/EPIC-07-analytics-setup.md)

## Later (post-MVP, not now)

Mattermost, Telegram, native mobile push, phone/SMS paging, status pages, runbook automation,
multi-tenant SaaS, self-hosted IdP (Keycloak, LDAP), local username/password auth, eXpress SmartApp
embedded view, scheduled chat digests, data retention/purge tooling, Helm/K8s deploy.
