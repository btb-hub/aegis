# Roadmap

Ordered so each phase ends with something demonstrable. The agent loop pulls stories top-down; finish
a phase's "exit" before moving on. Epics and their stories are in [`epics/`](./epics/).

## MVP status

**Phases 0–6 are complete** on `main` (latest: [PR #13](https://github.com/btb-hub/aegis/pull/13),
Phase 6). All stories in EPIC-01 through EPIC-07 are `Done`.

**Phases 0–7 are complete** on `main` ([PR #15](https://github.com/btb-hub/aegis/pull/15), Phase 7).

**Phases 0–9 are complete** on `main` (latest: [PR #21](https://github.com/btb-hub/aegis/pull/21), Phase 9).

| Phase | Epic | Merged |
|-------|------|--------|
| 0, 3.5 | [EPIC-01](./epics/EPIC-01-foundation.md) | (foundation PRs) |
| 1 | [EPIC-02](./epics/EPIC-02-shifts.md) | — |
| 2–3 | [EPIC-03](./epics/EPIC-03-integrations.md), [EPIC-04](./epics/EPIC-04-incidents.md) | — |
| 4 | [EPIC-05](./epics/EPIC-05-alerting.md) | PR #11 |
| 5 | [EPIC-06](./epics/EPIC-06-l2-l3.md) | PR #12 |
| 6 | [EPIC-07](./epics/EPIC-07-analytics-setup.md) | PR #13 |
| 7 | [EPIC-08](./epics/EPIC-08-dev-auth.md) | PR #15 |
| 8 | [EPIC-09](./epics/EPIC-09-teams-users-shifts.md) | PR #20 |
| 9 | [EPIC-10](./epics/EPIC-10-ui-polish.md) | PR #21 |
| 10 | [EPIC-11](./epics/EPIC-11-support-levels-workspaces.md) | — |
| 12 | [EPIC-13](./epics/EPIC-13-integration-admin.md) | — |
| 13 | [EPIC-14](./epics/EPIC-14-alert-ops-public-ingress.md) | — |

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

## Phase 8 — Teams, users & shifts setup *(Done — PR #20)*

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

**Next story:** none — Phase 8 complete.

## Phase 9 — UI polish *(Done — PR #21)*

*Exit: Alerts workspace matches the design-system filter bar; shared layout/table components exist;
every route uses consistent page headers, breadcrumbs, and spacing.*

**Why now:** Feature epics prioritised behaviour over composition. The Alerts page (and several others)
use misaligned ad-hoc layouts, raw native controls, and inconsistent typography relative to
[`docs/design_system.html`](./docs/design_system.html).

- Shared `Select`, `Banner`, `Checkbox`, `PageHeader`, `PageContent`, `DataTable`, `Pagination`
- Alerts page redesigned to the **Search & Filter Bar** pattern
- Cross-page header, breadcrumb, and content-width consistency
- Incidents, dashboard, teams, setup, account secondary polish

→ Epic: [EPIC-10 UI polish](./epics/EPIC-10-ui-polish.md)

**Next story:** none — Phase 9 complete.

## Phase 10 — Support levels, workspaces & incident wiring

*Exit: incidents page uses the real API (handoff persists on refresh); teams have L1/L2/L3/NOC tiers in
workspaces; admins configure escalation paths and routing rules; per-workspace Jira project keys work;
shared timeline unchanged (REQ-L2L3-03).*

**Why now:** Phase 5 shipped handoff APIs and Phase 8 wired teams/shifts, but incidents still run on
demo fixtures in `App.tsx` — **Send to L3** updates local state only. Teams have no support tier or
project grouping, so L2 and L3 are indistinguishable and handoff targets are hard-coded.

- Wire incidents list/detail to API (ack, resolve, handoff, bounce).
- Introduce **workspaces** (project scope within one deployment — not full multi-tenant SaaS).
- Team **support tiers** (L1 / L2 / L3 / NOC) and **escalation paths** with tier adjacency validation.
- **Per-workspace integrations** — Jira `project_key` override per workspace.
- **Routing rules UI** — workspace-scoped alert routing admin.
- **Shared timeline policy** — REQ-L2L3-03 unchanged; regression tests for all tiers.
- Admin UI + setup wizard for workspace, escalation, and routing configuration.

→ Epic: [EPIC-11 Support levels & workspaces](./epics/EPIC-11-support-levels-workspaces.md)

**Next story:** [AEG-078](./epics/EPIC-11-support-levels-workspaces.md) — Incidents page wired to API.

## Phase 12 — Integration admin configuration

*Exit: admins configure Jira, Slack, and eXpress credentials from `/integrations` alone (create + edit),
test connection returns actionable errors, secrets are redacted on list, and enable/disable/delete
work from the same page — without using the setup wizard.*

**Product decision:** Prefer **independent admin pages** over the multi-step setup wizard. Configuring
one connector must not require running earlier wizard steps. The wizard may stay as an optional
first-run checklist; it is not the configuration source of truth. See [EPIC-13](./epics/EPIC-13-integration-admin.md).

**Why now:** EPIC-03 delivered the connector implementations and test API. Useful credential fields
exist only on the wizard integrations step today, while `/integrations` stores kind/name with empty
`config`. Test connection fails with `integration provider is not configured` for Jira (same for Slack /
eXpress). Fix the dedicated Integrations page instead of routing admins through `/setup`.

- Validate required config on save; clarify test failures (AEG-093).
- Redact secrets on list; `PATCH /integrations/{id}` with secret keep-on-omit (AEG-094).
- Integrations create/edit UI with provider fields (AEG-095).
- Enable/disable, delete, incomplete-state messaging (AEG-096).

→ Epic: [EPIC-13 Integration admin](./epics/EPIC-13-integration-admin.md)

**Next story:** [AEG-093](./epics/EPIC-13-integration-admin.md) — Validate integration config and clarify
test failures.

## Phase 13 — Alert operations and public chat ingress

*Exit: an admin finds routing from Alerts; an operator can assign, create an incident, or resolve from
an alert row; BotX/Slack/webhook traffic is documented to skip Google IAP (or use a second public
host), with copyable callback URLs on Integrations.*

**Why now:** Colleagues received test alerts and could not find **Configure routing** (it lives on
workspace detail). Alert rows are read-only — assign / create incident / resolve exist only on
incidents, and incidents never appear when no rule matches. eXpress BotX cannot call
`PUBLIC_URL` if interactive Google auth/IAP wraps the same origin; a reverse proxy is the intended
workaround and must be a first-class deploy path.

- Admin **Configure routing** on `/alerts` → `/workspaces` (AEG-097). No new Routing nav.
- Alert list `incident_id`; manual `POST /incidents`; assign; unlinked alert resolve (AEG-098–101).
- Alert row actions UI (AEG-102).
- Public ingress docs + copyable BotX/Slack URLs (AEG-103).

→ Epic: [EPIC-14 Alert operations & public ingress](./epics/EPIC-14-alert-ops-public-ingress.md)

**Next story:** [AEG-097](./epics/EPIC-14-alert-ops-public-ingress.md) — Configure routing CTA on Alerts.

## Later (post-MVP, not now)

Mattermost, Telegram, native mobile push, phone/SMS paging, status pages, runbook automation,
multi-tenant SaaS, self-hosted IdP (Keycloak, LDAP), local username/password auth, eXpress SmartApp
embedded view, scheduled chat digests, data retention/purge tooling, Helm/K8s deploy.
