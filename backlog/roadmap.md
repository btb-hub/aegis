# Roadmap

Ordered so each phase ends with something demonstrable. The agent loop pulls stories top-down; finish
a phase's "exit" before moving on. Epics and their stories are in [`epics/`](./epics/).

## MVP status

**Phases 0–6 are complete** on `main` (latest: [PR #13](https://github.com/btb-hub/aegis/pull/13),
Phase 6). All stories in EPIC-01 through EPIC-07 are `Done`.

**Phases 0–7 are complete** on `main` ([PR #15](https://github.com/btb-hub/aegis/pull/15), Phase 7).

| Phase | Epic | Merged |
|-------|------|--------|
| 0, 3.5 | [EPIC-01](./epics/EPIC-01-foundation.md) | (foundation PRs) |
| 1 | [EPIC-02](./epics/EPIC-02-shifts.md) | — |
| 2–3 | [EPIC-03](./epics/EPIC-03-integrations.md), [EPIC-04](./epics/EPIC-04-incidents.md) | — |
| 4 | [EPIC-05](./epics/EPIC-05-alerting.md) | PR #11 |
| 5 | [EPIC-06](./epics/EPIC-06-l2-l3.md) | PR #12 |
| 6 | [EPIC-07](./epics/EPIC-07-analytics-setup.md) | PR #13 |
| 7 | [EPIC-08](./epics/EPIC-08-dev-auth.md) | PR #15 |
| 8 | [EPIC-09](./epics/EPIC-09-teams-users-shifts.md) | — |

## Guiding order

Build a **thin vertical slice first** (one alert can become one incident with one connector and you
can see who's on call), then thicken it. Don't build all the models, then all the APIs — build
features end to end.

## Phase 0 — Foundation *(Done)*

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

## Phase 1 — Shifts & on-call *(Done)*

*Exit: admins define teams + a rotation; "who's on call now" is correct, including DST + an override.*

**Gate:** Phase 0 Storybook story ([AEG-056](./epics/EPIC-01-foundation.md)) must be `Done` before picking EPIC-02 stories.

- Teams, memberships, schedules, overrides, on-call resolution + materialisation.
- Calendar API/view and "who's on call now" view.

→ Epic: [EPIC-02 Shifts](./epics/EPIC-02-shifts.md)

## Phase 2 — Alert → incident → first connector → Jira (the spine) *(Done)*

*Exit: a firing alert creates one incident, makes a Jira ticket, and pages the on-call on Slack; ack works.*

- Alert processing: dedup/group, routing rules, incident creation + lifecycle, timeline.
- Jira ticket provider (create + link).
- Slack chat connector end to end with Acknowledge.
- Escalation timer + one escalation step.

→ Epics: [EPIC-04 Incidents](./epics/EPIC-04-incidents.md), [EPIC-03 Integrations](./epics/EPIC-03-integrations.md)

## Phase 3 — eXpress connector *(Done)*

*Exit: notifications fan out to Slack and eXpress; each has Test connection + Acknowledge.*

- eXpress via BotX HTTP API (HMAC token, bot endpoints, bubbles, `/link` bootstrap).
- Per-connector Test connection; graceful degradation + retry/backoff.

→ Epic: [EPIC-03 Integrations](./epics/EPIC-03-integrations.md)

## Phase 3.5 — Web auth & session *(Done)*

*Exit: a user signs in from the web UI (Google, Slack, or eXpress); the session persists; API-backed
pages (integrations, later shifts/incidents) work without typing auth URLs manually.*

**Gate:** Phase 3.5 must be `Done` before picking EPIC-05 UI stories ([AEG-038](./epics/EPIC-05-alerting.md), [AEG-039](./epics/EPIC-05-alerting.md)).

- Login page with OIDC provider buttons (reuses API from [AEG-005](./epics/EPIC-01-foundation.md)).
- App shell session: `GET /auth/me`, sign out, redirect unsigned users from protected routes.
- OIDC callback redirects browser back to the web app after login.

→ Epic: [EPIC-01 Foundation](./epics/EPIC-01-foundation.md) ([AEG-057](./epics/EPIC-01-foundation.md)–[AEG-059](./epics/EPIC-01-foundation.md))

## Phase 4 — Alerting workspace *(Done — PR #11)*

*Exit: filter, search, group, saved views, inline analytics, CSV export — fast over large volumes.*

- Indexed list (labels GIN, search tsv), filters, grouping, saved views, inline analytics + export.

→ Epic: [EPIC-05 Alerting](./epics/EPIC-05-alerting.md)

## Phase 5 — L2 ↔ L3 transparency *(Done — PR #12)*

*Exit: L2 hands an incident to the right L3 on-call in one action; both see one shared timeline;
handoff analytics work.*

- Handoff action + record, shared incident view, reassign/bounce, handoff analytics.

→ Epic: [EPIC-06 L2↔L3](./epics/EPIC-06-l2-l3.md)

## Phase 6 — Analytics & polish *(Done — PR #13)*

*Exit: the five north-star questions answered on one dashboard; setup wizard end to end; docs match.*

- Overview dashboard (MTTA/MTTR/escalation/noise/load), drill-down, compare-to-previous.
- Setup wizard incl. "Send test alert"; accessibility pass; setup-time check against NFR-1.

→ Epic: [EPIC-07 Analytics & Setup](./epics/EPIC-07-analytics-setup.md)

## Phase 7 — Local dev auth *(Done — PR #15)*

*Exit: protected pages work on localhost without OIDC app registration; production auth unchanged.*

- Opt-in `DEV_AUTH_ENABLED` dev login (real session + RBAC).
- Login page **Dev sign in** when enabled; docs for local testing.

→ Epic: [EPIC-08 Local dev auth](./epics/EPIC-08-dev-auth.md)

**Next story:** none on Phase 7 — see Phase 8 below.

## Phase 8 — Teams, users & shifts setup *(Next — this branch)*

*Exit: admin creates a team, adds SSO-like users, defines a rotation and overrides in the UI; shifts
calendar shows live on-call data; local dev can seed realistic users without OIDC.*

**Why now:** Phase 1 backend for teams/schedules is done, but the web app still uses demo fixtures.
Dev auth (Phase 7) unlocks protected pages yet cannot configure on-call. SSO upsert exists but OIDC
userinfo is still stubbed; there is no user directory API or dev seeds.

- Real OIDC profile provisioning with multi-provider **fill-if-empty backfill** (`user_identities`).
- Account page (profile, locale, connected providers, eXpress link).
- Admin users list + dev seed command.
- Teams CRUD/members UI; shifts page wired to API; schedule/override admin UI.
- Setup wizard team step; docs updated.

→ Epic: [EPIC-09 Teams, users & shifts setup](./epics/EPIC-09-teams-users-shifts.md)

**Next story:** [AEG-066](./epics/EPIC-09-teams-users-shifts.md) (dev user seeds — in progress).

## Later (post-MVP, not now)

Mattermost, Telegram, native mobile push, phone/SMS paging, status pages, runbook automation,
multi-tenant SaaS, self-hosted IdP (Keycloak, LDAP), local username/password auth, eXpress SmartApp
embedded view, scheduled chat digests, data retention/purge tooling, Helm/K8s deploy.
