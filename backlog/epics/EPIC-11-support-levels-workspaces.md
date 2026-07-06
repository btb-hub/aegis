# EPIC-11 — Support levels, workspaces & incident wiring

**Phase:** 10  
**Exit:** Incidents page uses the real API (handoff persists on refresh); teams have L1/L2/L3/NOC
tiers grouped in workspaces; admins configure escalation paths and routing rules; handoff/escalate
picker shows only valid targets; per-workspace Jira project keys work; shared timeline unchanged
(REQ-L2L3-03).

## Problem

Phase 5 shipped **backend** L2↔L3 handoff ([EPIC-06](./EPIC-06-l2-l3.md)). Phase 8 wired teams
and shifts to the API ([EPIC-09](./EPIC-09-teams-users-shifts.md)) but explicitly left **incidents
on demo fixtures** in `App.tsx`. Users clicking **Send to L3** see a timeline row that vanishes on
refresh because no API call is made.

There is also **no domain model** for support levels: all teams are flat, handoff targets are
hard-coded (`Platform L3`, `Data L3`), and nothing groups teams by business project. Real
organisations run multiple projects (Platform, Data, Payments), each with an L2/L3 pair and distinct
routing — or occasionally share an L3 pool.

Full **multi-tenant SaaS** (separate orgs per project) is a post-MVP non-goal. This epic introduces
**workspaces** as light-weight project scope within one deployment. See the design spec:
[`docs/features/support-levels-and-workspaces.md`](../docs/features/support-levels-and-workspaces.md).

## Solution (seven tracks)

1. **Incident API wiring** — replace `App.tsx` fixtures; TanStack Query; ack/resolve/handoff/bounce
   call existing endpoints.
2. **Workspaces** — top-level project grouping; teams belong to a workspace.
3. **Support tiers & escalation paths** — `support_tier` on teams (`l1`, `l2`, `l3`, `noc`); tier-aware
   path configuration; handoff validation.
4. **Admin UI & wizard** — tier badges, path editor, setup step.
5. **Per-workspace integrations** — Jira `project_key` (and optional Slack channel) scoped to workspace.
6. **Routing rules UI** — workspace-scoped alert routing admin (labels → team).
7. **Timeline policy** — REQ-L2L3-03 unchanged: all tiers see identical timeline; regression tests only
   (no role-based filtering).

---

### AEG-078 — Incidents page wired to API

- **Status:** Ready
- **Depends on:** AEG-056, EPIC-04 (incident APIs), EPIC-06 (handoff APIs)
- **PRD:** REQ-SLV-04; REQ-INC-05, REQ-L2L3-01
- **Acceptance:**
  - [ ] Remove `initialIncidents`, `handoffTeams`, and mutation handlers from `App.tsx`
  - [ ] `IncidentsPage` fetches `GET /incidents` and `GET /incidents/{id}` via TanStack Query
  - [ ] Acknowledge, resolve, handoff, bounce call respective POST endpoints and invalidate queries
  - [ ] Timeline events persist after page refresh (integration test or e2e-style web test)
  - [ ] Loading and error states use shared `Banner` / skeleton patterns
  - [ ] `IncidentsPage.test.tsx` updated; no regression in `App.test.tsx`
  - [ ] Document in [`docs/features/incident-management.md`](../docs/features/incident-management.md)

**Plan:** Add `apps/web/src/lib/incidentsApi.ts`. Temporary handoff picker: all teams until AEG-081.
Branch: `feat/incidents-AEG-078-api-wiring`.

---

### AEG-079 — Workspaces model and API

- **Status:** Ready
- **Depends on:** AEG-067
- **PRD:** REQ-SLV-01
- **Acceptance:**
  - [ ] Migration: `workspaces` table; `teams.workspace_id` FK; backfill single `Default` workspace
  - [ ] CRUD API: `GET/POST/PATCH/DELETE /workspaces`
  - [ ] Team create/update requires `workspace_id`; list teams filterable by workspace
  - [ ] Unit tests for service + handler; migration up/down
  - [ ] Update [`docs/03-data-model.md`](../docs/03-data-model.md) and [`docs/04-api-spec.md`](../docs/04-api-spec.md)

**Plan:** `WorkspaceService` in `apps/api/internal/service/`. Branch: `feat/workspaces-AEG-079`.

---

### AEG-080 — Team support tier (L1, L2, L3, NOC)

- **Status:** Ready
- **Depends on:** AEG-079
- **PRD:** REQ-SLV-02, REQ-SLV-06, REQ-SLV-08
- **Acceptance:**
  - [ ] Migration: `teams.support_tier` nullable enum (`l1`, `l2`, `l3`, `noc`)
  - [ ] Team API exposes and accepts `support_tier`; validate allowed values
  - [ ] Teams list UI: workspace column + tier badge (L1 / L2 / L3 / NOC / —)
  - [ ] Team detail: tier selector on create/edit with short helper text per tier
  - [ ] Locale strings `en` + `ru` for tier labels and descriptions
  - [ ] Tests for API validation and UI render

**Plan:** Extend existing team handler/service. Branch: `feat/teams-AEG-080-support-tier`.

---

### AEG-081 — Escalation paths and handoff validation

- **Status:** Ready
- **Depends on:** AEG-080, AEG-040 (handoff service)
- **PRD:** REQ-SLV-03, REQ-SLV-05, REQ-SLV-08
- **Acceptance:**
  - [ ] Migration: `escalation_paths` (`from_team_id`, `to_team_id`, `workspace_id`, `cross_workspace`)
  - [ ] API: `GET/PUT /workspaces/{id}/escalation-paths`
  - [ ] `GET /teams/{id}/handoff-targets` returns allowed target teams for the owning team's tier
  - [ ] Path validation enforces tier adjacency: `noc→l1`, `noc→l2`, `l1→l2`, `l2→l3` (reject invalid pairs)
  - [ ] `HandoffService` rejects disallowed `to_team_id` with structured error
  - [ ] Unit tests: allowed paths per tier pair succeed; disallowed returns 400
  - [ ] Update API spec and feature doc with validation rules

**Plan:** Path PUT replaces full set (idempotent admin edit). Branch: `feat/escalation-AEG-081-paths`.

---

### AEG-082 — Escalation UI uses paths (L1/L2/L3/NOC)

- **Status:** Ready
- **Depends on:** AEG-078, AEG-081
- **PRD:** REQ-SLV-05, REQ-SLV-08, REQ-L2L3-01
- **Acceptance:**
  - [ ] Incident detail fetches targets from `GET /teams/{owningTeamId}/handoff-targets`
  - [ ] Escalate button label is contextual by owning tier: **Escalate to L2**, **Hand off to L3**,
        **Escalate to L1** (from NOC) — locale keys per pair
  - [ ] Picker hidden or disabled when no paths configured (helper text: configure in Teams)
  - [ ] Incident detail shows owning team name + tier; assignee team after escalation
  - [ ] Bounce button visible when latest non-bounced handoff exists; label **Bounce to L2** / **Bounce to L1**
        as appropriate
  - [ ] Web tests: escalation calls API; picker lists only configured targets

**Plan:** Depends on AEG-078 for API-backed incidents. Branch: `feat/incidents-AEG-082-escalation-ui`.

---

### AEG-083 — Escalation path admin UI

- **Status:** Ready
- **Depends on:** AEG-081, AEG-080
- **PRD:** REQ-SLV-06, REQ-SLV-08
- **Acceptance:**
  - [ ] Team detail: section **Escalation paths** — add/remove allowed target teams (filtered by valid tier)
  - [ ] Team detail: section **Escalated from** — read-only incoming paths
  - [ ] Workspace filter on Teams page (optional dropdown)
  - [ ] Empty state when team has no outgoing paths; tier-specific guidance (e.g. L2 needs an L3 target)
  - [ ] Storybook story if new shared multi-select component; else use existing `Select`
  - [ ] Tests for path add/remove flow

**Plan:** Reuse team detail page from AEG-067. Branch: `feat/teams-AEG-083-path-admin-ui`.

---

### AEG-084 — Setup wizard workspace & escalation step

- **Status:** Ready
- **Depends on:** AEG-079, AEG-083, AEG-070
- **PRD:** REQ-SLV-07
- **Acceptance:**
  - [ ] New wizard step (or extend team step): create workspace, L2 team, L3 team, default L2→L3 path
  - [ ] Optional: add L1 team + L1→L2 path (collapsed "Advanced" section)
  - [ ] Skip step if workspace with escalation paths already exists
  - [ ] Update setup docs [`docs/07-setup-deployment.md`](../docs/07-setup-deployment.md)

**Plan:** Minimal happy path — one workspace, one L2/L3 pair, one path. Branch: `feat/setup-AEG-084-wizard-workspaces`.

---

### AEG-085 — Per-workspace integrations

- **Status:** Ready
- **Depends on:** AEG-079, AEG-003 (integrations)
- **PRD:** REQ-SLV-09
- **Acceptance:**
  - [ ] Migration: `integrations.workspace_id` nullable FK; NULL = global fallback
  - [ ] Workspace Jira integration: `config.project_key` required when `workspace_id` set; inherits
        `base_url`, credentials from global Jira integration when present
  - [ ] Worker resolves integration: incident → owning team → workspace → workspace integration, else global
  - [ ] Optional: workspace Slack override (`config.channel_id`) for paging
  - [ ] Integrations UI: workspace selector when adding/editing; show scope badge (Global / Workspace)
  - [ ] Setup wizard: optional workspace Jira project key on workspace step (or integrations sub-step)
  - [ ] Unit tests: ticket creation uses workspace project key; fallback to global
  - [ ] Update [`docs/03-data-model.md`](../docs/03-data-model.md), [`docs/integrations/jira.md`](../docs/integrations/jira.md)

**Plan:** No duplicate credentials — workspace rows override project/channel only. Branch: `feat/integrations-AEG-085-workspace-scope`.

---

### AEG-086 — Routing rules UI

- **Status:** Ready
- **Depends on:** AEG-079, AEG-067, EPIC-04 (routing rules API)
- **PRD:** REQ-SLV-10, REQ-INC-04
- **Acceptance:**
  - [ ] Workspace detail page (or team detail section): **Routing rules** list sorted by priority
  - [ ] Create/edit rule: label matchers (key=value rows), target team (same workspace), priority
  - [ ] Delete rule with confirm
  - [ ] Calls existing `GET/POST/PATCH/DELETE /routing-rules` API; filter display by workspace teams
  - [ ] Empty state: link to docs + "Add rule" when no rules
  - [ ] Setup wizard: optional first routing rule (e.g. `project=<slug>` → L2 team)
  - [ ] Locale strings `en` + `ru`; web tests for CRUD flow

**Plan:** New `WorkspaceDetailPage` or extend `TeamsPage` with workspace tabs. Branch: `feat/routing-AEG-086-rules-ui`.

---

### AEG-087 — Shared timeline policy (REQ-L2L3-03 unchanged)

- **Status:** Ready
- **Depends on:** AEG-078, AEG-082
- **PRD:** REQ-L2L3-03, REQ-SLV-11
- **Acceptance:**
  - [ ] Document policy in [`docs/features/l2-l3-transparency.md`](../docs/features/l2-l3-transparency.md):
        all support tiers see identical `timeline_events`; no role/tier filtering in API or UI
  - [ ] API: `GET /incidents/{id}/timeline` returns full event list for any authenticated member with
        incident access (no new filtering logic)
  - [ ] Regression test (Go + web): L1, L2, and L3 team members receive the same event count and kinds
  - [ ] UI: no "internal only" event types introduced in this epic
  - [ ] Explicit non-goal note: future private comments are out of scope for Phase 10

**Plan:** Policy + tests only — no timeline filtering feature. Branch: `feat/timeline-AEG-087-shared-visibility`.

---

## Dependency graph

```
EPIC-04/06 (incident + handoff APIs — Done)
  └── AEG-078 (incidents UI wiring)
AEG-067 (teams UI — Done)
  └── AEG-079 (workspaces)
        ├── AEG-080 (support tier: l1/l2/l3/noc)
        │     └── AEG-081 (escalation paths + validation)
        │           ├── AEG-082 (escalation UI)
        │           └── AEG-083 (path admin UI)
        ├── AEG-085 (per-workspace integrations)
        └── AEG-086 (routing rules UI)
AEG-070 (wizard — Done) + AEG-083 + AEG-085 → AEG-084 (wizard step)
AEG-078 + AEG-082 → AEG-087 (shared timeline policy + tests)
```

**Suggested implementation order:** AEG-078 first (fixes the refresh bug), then AEG-079 → AEG-080 →
AEG-081; AEG-085 and AEG-086 can run in parallel once AEG-079 lands; AEG-087 last.

---

## Out of scope

- Multi-tenant SaaS (`tenant_id` on all tables, separate auth)
- Per-workspace OIDC or separate credentials per workspace (workspace integrations inherit global creds)
- Role-based timeline filtering or "internal only" events (REQ-L2L3-03 stays shared)
- Private comments / tier-restricted notes (future epic)

---

## Definition of done (epic)

- [ ] User escalations persist after refresh
- [ ] Teams show L1/L2/L3/NOC tier and workspace
- [ ] Escalation only allowed to configured paths per tier adjacency rules
- [ ] Per-workspace Jira project keys used for ticket create/assignee update
- [ ] Routing rules manageable in UI per workspace
- [ ] All tiers see identical timeline (REQ-L2L3-03 regression tests pass)
- [ ] Setup wizard configures a workspace + L2/L3 pair (+ optional L1)
- [ ] `make lint type test` green; API spec and data model docs updated
- [ ] Feature spec [`support-levels-and-workspaces.md`](../docs/features/support-levels-and-workspaces.md) reflects shipped behaviour
