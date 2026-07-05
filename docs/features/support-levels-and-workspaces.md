# Feature: Support levels, workspaces & L2↔L3 escalation

Design spec for Phase 10 ([EPIC-11](../../backlog/epics/EPIC-11-support-levels-workspaces.md)).
Extends [L2↔L3 transparency](./l2-l3-transparency.md) (Phase 5) with a real domain model and
closes the gap between backend handoff APIs and the web UI.

## Problem

### What users see today

1. **Handoff does not persist.** Clicking **Send to L3** appends a timeline row in the browser, but
   refreshing the page removes it. Acknowledge and resolve behave the same way.
2. **L2 and L3 teams are indistinguishable.** The Teams page treats every team the same — no tier,
   badge, or escalation relationship. The incident handoff picker uses two hard-coded demo teams
   (`Platform L3`, `Data L3`) in `App.tsx`, not teams from the database.
3. **No project scope.** A single deployment may serve multiple business projects (Platform, Data,
   Payments), each with its own L2/L3 pair, Jira project, and alert labels — but the data model has
   no way to group teams or route within a project boundary.

### Root cause (technical)

| Layer | State |
|-------|-------|
| **Backend** | Handoff, bounce, acknowledge, resolve, and timeline APIs exist and persist to Postgres ([EPIC-06](../../backlog/epics/EPIC-06-l2-l3.md), [EPIC-04](../../backlog/epics/EPIC-04-incidents.md)). |
| **Frontend** | `IncidentsPage` is still fed from in-memory fixtures in `App.tsx`. Mutations update React state only; no API calls. Explicitly deferred in [EPIC-09](../../backlog/epics/EPIC-09-teams-users-shifts.md). |
| **Teams model** | `teams` has `name`, `description` only — no support tier, no parent project, no escalation path. |
| **Integrations** | One global Jira config with a single `project_key`; no per-project override. |

The product *looks* like L2↔L3 works (buttons, timeline UI, analytics widgets) but the incident
workspace is a demo shell on top of a real backend.

---

## Goals

1. **Persist incident actions** — handoff, bounce, ack, resolve survive refresh and match the DB.
2. **Model support tiers** — teams are explicitly L1, L2, L3, or NOC (or unset for admin-only groups).
3. **Configure escalation paths** — tier-aware paths (e.g. L1→L2, L2→L3, NOC→L1/L2).
4. **Scope by project** — group teams and routing within a **workspace** (project) without full
   multi-tenant SaaS complexity.
5. **Make escalation intentional** — picker shows only valid targets for the incident's owning team.
6. **Per-workspace integrations** — Jira tickets land in the correct project per workspace.
7. **Routing rules in UI** — admins configure alert label → team routing without API calls.
8. **Shared timeline** — REQ-L2L3-03 unchanged: all tiers see identical events (no role filtering).

## Non-goals (this phase)

- Full **multi-tenant SaaS** (separate orgs, billing, cross-tenant isolation) — see
  [Decision: workspace vs tenant](#decision-workspace-vs-tenant).
- Per-workspace OIDC or separate auth realms.
- **Timeline filtering by role or tier** — explicitly rejected; see
  [Timeline visibility policy](#timeline-visibility-policy-req-l2l3-03-unchanged).
- Private / tier-restricted comments on timeline (future epic).
- Duplicate integration credentials per workspace (workspace rows override project/channel only).

---

## Personas & scenarios

### Scenario A — Single project, one L2/L3 pair

Platform alerts (`project=platform`) route to **Platform L2**. On-call L2 engineer investigates,
clicks **Hand off to L3**, picks **Platform L3** (the only configured target). L3 on-call is paged;
timeline shows the handoff; analytics record response time.

### Scenario B — Multiple projects in one deployment

| Workspace | L2 team | L3 team | Jira project |
|-----------|---------|---------|--------------|
| Platform | Platform L2 | Platform L3 | OPS |
| Data | Data L2 | Data L3 | DATA |

Alert with `project=data` routes to Data L2. Handoff picker offers **Data L3** only — not Platform L3.

### Scenario C — Shared L3 pool (optional path)

Data L2 may hand off to **Platform L3** when Data L3 is unavailable. Admin configures an extra
escalation path (`Data L2 → Platform L3`). Handoff API validates against configured paths.

### Scenario D — Cross-workspace visibility

IT manager sees incidents and handoff stats across all workspaces. L2 engineer on Platform L2 sees
Platform-scoped incidents by default; filter can widen if they are also a member of other teams.

### Scenario E — NOC triage → L1 → L2 → L3

Monitoring alert routes to **Platform NOC** (tier `noc`). NOC on-call triages and **Escalates to L1**
(Platform L1 helpdesk). L1 resolves common issues or **Escalates to L2**. L2 **Hands off to L3** for
deep infra work. Each step uses configured escalation paths; timeline shows the full chain to all tiers.

### Scenario F — Per-workspace Jira

Platform workspace uses Jira project `OPS`; Data workspace uses `DATA`. When an incident is created for
a Platform L2 team, the worker creates the ticket in `OPS`. After handoff to Platform L3, assignee
update still uses the Platform workspace integration.

---

## Decision: workspace vs tenant

The user asked whether **different tenants per project** would be better.

| Approach | Fits when | Pros | Cons |
|----------|-----------|------|------|
| **Workspace (recommended)** | One company, multiple projects/BUs sharing SSO and admins | Simple migration; one login; shared user directory; cross-project analytics; matches [product brief](../00-product-brief.md) single-org scope | Weaker isolation; one bad config affects all workspaces |
| **Tenant per project** | Selling Aegis as SaaS or hard regulatory isolation between BUs | Strong data/integration isolation; separate Jira/Slack per tenant | Heavy lift: tenant_id on every table, auth scoping, per-tenant integrations, no shared users without federation — listed as post-MVP non-goal |
| **Team tier only (minimal)** | One project today, defer grouping | Smallest change | Does not solve multi-project routing or handoff picker scoping |

**Recommendation:** Introduce **`workspaces`** as a light-weight project scope inside one deployment.
Reserve **multi-tenant SaaS** for a later ADR if the product pivots to external customers.

Naming: **workspace** in code/API (avoids collision with Jira "project"); **Project** is fine in
admin UI copy ("Platform project").

---

## Domain model

### New entities

#### `workspaces`

Top-level project scope within one Aegis deployment.

| Column | Type | Notes |
|--------|------|-------|
| id | uuid PK | |
| name | text | Display name, e.g. "Platform" |
| slug | text | Unique, URL-safe, e.g. `platform` |
| description | text | Optional |
| created_at | timestamptz | |
| updated_at | timestamptz | |

Migration backfill: one workspace `Default` (slug `default`); existing teams assigned to it.

#### `escalation_paths`

Directed escalation allowance between teams. Tier adjacency is validated on write.

| Column | Type | Notes |
|--------|------|-------|
| id | uuid PK | |
| from_team_id | uuid FK → teams | Source team |
| to_team_id | uuid FK → teams | Target team |
| workspace_id | uuid FK → workspaces | Denormalised for query; both teams must belong to same workspace unless `cross_workspace` |
| cross_workspace | bool | Default false; when true, allows escalation across workspace boundary (Scenario C) |
| created_at | timestamptz | |

Unique on `(from_team_id, to_team_id)`.

**Allowed tier pairs** (application validation):

| From tier | To tier | Typical use |
|-----------|---------|-------------|
| `noc` | `l1` | NOC triage → first-line helpdesk |
| `noc` | `l2` | NOC bypass L1 for known infra alerts |
| `l1` | `l2` | Helpdesk → product/engineering L2 |
| `l2` | `l3` | L2 → deep technical (existing handoff) |

Reverse bounce uses the latest non-bounced handoff record (unchanged from Phase 5).

### Extended entities

#### `teams` (add columns)

| Column | Type | Notes |
|--------|------|-------|
| workspace_id | uuid FK → workspaces | Required after migration |
| support_tier | text nullable | `l1`, `l2`, `l3`, `noc`, or NULL (general/admin team) |

**Tier definitions:**

| Tier | Role | Typical paging |
|------|------|----------------|
| `noc` | Network/monitoring operations centre; first human eyes on automated alerts | 24/7 rotation |
| `l1` | First-line helpdesk; password resets, known issues, initial triage | Business-hours or follow-the-sun |
| `l2` | Product/application support; owns incident until resolved or escalated | On-call rotation |
| `l3` | Deep technical / platform engineering | On-call rotation |

Constraints (application-level):

- A team with an outgoing escalation tier should have at least one valid path (warn in wizard).
- Escalation target must appear in `escalation_paths` for the incident's owning team.
- `incidents.team_id` remains the **owning team at creation** (usually L1 or L2); does not change on escalation.

#### `integrations` (extend)

| Column | Type | Notes |
|--------|------|-------|
| workspace_id | uuid FK nullable | NULL = global default; set for workspace-specific overrides |

**Resolution order** when worker needs an integration for an incident:

1. Look up incident → owning team → workspace.
2. Use enabled integration where `kind = jira` AND `workspace_id = <workspace>`.
3. If none, fall back to global integration (`workspace_id IS NULL`).

Workspace Jira row stores at minimum `project_key`; inherits `base_url`, `email`, `api_token` from
global Jira config. Optional workspace Slack row stores `channel_id` override for paging.

### Relationships

```text
workspaces ──< teams ──< schedules, overrides, routing_rules
workspaces ──< integrations (optional overrides)
teams ──< escalation_paths >── teams
teams ──< incidents
incidents ──< handoffs, timeline_events
```

### Incident ownership semantics

| Field | Meaning after this feature |
|-------|----------------------------|
| `incidents.team_id` | **Owning team** — the L2 team that received the alert via routing (unchanged). Does not change on handoff. |
| `incidents.assignee_id` | **Current handler** — L2 on-call at creation; L3 on-call after handoff; prior L2 on handoff bounce. |
| `handoffs.from_team_id` / `to_team_id` | Record L2 source and L3 target for analytics. |

UI shows: "Owned by Platform L2 · Assigned to \<user\> (Platform L3)" after handoff.

---

## User flows

### Admin: set up a project

1. Create workspace **Platform** (or use Default during migration).
2. Configure global Jira integration (credentials) if not already done.
3. Add workspace Jira override with `project_key = OPS`.
4. Create teams: **Platform NOC** (`noc`), **Platform L1** (`l1`), **Platform L2** (`l2`), **Platform L3** (`l3`) — minimum viable: L2 + L3 only.
5. Add escalation paths: NOC→L1, L1→L2, L2→L3 (configure only the pairs you use).
6. Add routing rules via UI: e.g. `project=platform` → Platform L2 (or NOC if alerts land there first).
7. Configure schedules for on-call teams ([EPIC-09](../../backlog/epics/EPIC-09-teams-users-shifts.md)).

Setup wizard gains a **Workspaces & escalation** step (or extends the team step).

### L2: hand off to L3

1. Open incident owned by Platform L2.
2. Click **Hand off to L3**.
3. Picker lists only teams from `escalation_paths` where `from_team_id = Platform L2`.
4. Optional note → `POST /incidents/{id}/handoff`.
5. Timeline reloads from API; handoff row persists.

### L1 / NOC: escalate

Same flow with contextual labels: **Escalate to L2** (from L1), **Escalate to L1** or **Escalate to L2**
(from NOC). Same API endpoint; validation uses tier adjacency rules.

### L3: bounce

Unchanged from [l2-l3-transparency.md](./l2-l3-transparency.md): required note, reassign to prior owner
from latest non-bounced handoff. UI label **Bounce to L2** (or **Bounce to L1** when bounced from L1→L2).

---

## Timeline visibility policy (REQ-L2L3-03 unchanged)

Phase 5 established that L2 and L3 see **identical** timeline events. Phase 10 extends this to **all
support tiers** — no change to the requirement:

| Principle | Detail |
|-----------|--------|
| **No tier filtering** | API and UI return the full `timeline_events` list for every user with incident access. |
| **No "internal" events** | Do not introduce event kinds visible only to L3 or NOC in this epic. |
| **Audit & trust** | L3 needs L2 context; L2 needs NOC triage notes; managers need the full chain. |
| **Bounce note is visible** | Bounce reason appears in timeline payload for all viewers (existing behaviour). |

**Explicit non-goal:** role-based or tier-based timeline views. If private notes are needed later, they
require a new PRD requirement and a separate epic (different event visibility model).

Regression coverage: [AEG-087](../../backlog/epics/EPIC-11-support-levels-workspaces.md) adds tests
that L1, L2, and L3 members receive identical event sets from the API.

---

## API changes

### Incidents (wire UI — no schema change)

| Method | Path | Notes |
|--------|------|-------|
| GET | `/incidents` | List with filters (`status`, `team_id`, `workspace_id`) |
| GET | `/incidents/{id}` | Detail + timeline + linked alerts |
| POST | `/incidents/{id}/acknowledge` | |
| POST | `/incidents/{id}/resolve` | |
| POST | `/incidents/{id}/handoff` | Validate `to_team_id` against escalation paths |
| POST | `/incidents/{id}/bounce` | |

### Workspaces

| Method | Path | Notes |
|--------|------|-------|
| GET | `/workspaces` | List |
| POST | `/workspaces` | Admin |
| GET | `/workspaces/{id}` | |
| PATCH | `/workspaces/{id}` | |
| DELETE | `/workspaces/{id}` | Only if no teams |

### Teams (extend)

- `GET/POST/PATCH /teams` include `workspace_id`, `support_tier`.
- `GET /teams/{id}/handoff-targets` — L3 teams reachable from this L2 team (for incident picker).

### Escalation paths

| Method | Path | Notes |
|--------|------|-------|
| GET | `/workspaces/{id}/escalation-paths` | |
| PUT | `/workspaces/{id}/escalation-paths` | Replace set (admin) |

Handoff service change: reject `to_team_id` not in allowed paths with `400` + structured error
`handoff_target_not_allowed`.

### Integrations (extend)

| Method | Path | Notes |
|--------|------|-------|
| GET | `/integrations` | Include `workspace_id` and scope in response |
| POST | `/integrations` | Accept optional `workspace_id`; workspace Jira requires `project_key` |
| PATCH | `/integrations/{id}` | Update workspace override config |

Worker change: resolve integration by incident workspace before global fallback.

### Routing rules (UI consumes existing API)

| Method | Path | Notes |
|--------|------|-------|
| GET | `/routing-rules` | UI filters by teams in selected workspace |
| POST | `/routing-rules` | Target team must belong to workspace |
| PATCH | `/routing-rules/{id}` | |
| DELETE | `/routing-rules/{id}` | |

Optional future: add `routing_rules.workspace_id` for direct scoping; Phase 10 can filter via team
membership in workspace.

---

## UI changes

| Surface | Change |
|---------|--------|
| **Incidents** | TanStack Query fetch; mutations call API; remove `App.tsx` fixtures |
| **Incident detail** | Owner team + tier badge; assignee with team name; contextual escalate/bounce labels |
| **Teams list** | Columns: workspace, support tier (L1 / L2 / L3 / NOC / —) |
| **Team detail** | Edit tier; configure escalation paths (outgoing + incoming) |
| **Workspace detail** | Routing rules CRUD; optional integrations tab |
| **Integrations** | Scope badge (Global / Workspace); workspace selector |
| **Setup wizard** | Workspace + tiers + paths + optional routing rule + Jira project key |
| **Dashboard** | Optional workspace filter on widgets (stretch) |

Copy: contextual escalation labels per tier pair ([writing rules](../11-localization.md)) — e.g.
**Escalate to L2**, **Hand off to L3**, **Bounce to L2**.

---

## Migration & rollout

1. Create `workspaces`; insert `Default`; set all existing `teams.workspace_id`.
2. Add `teams.support_tier` nullable.
3. Create `escalation_paths`.
4. Seed script / wizard prompt: admin sets tiers and paths for existing teams.
5. Deploy API validation; deploy UI wiring.

No breaking change to existing handoff rows. Teams without tier behave as today (handoff allowed to
any team until paths configured — or strict mode after cutover; **recommend strict mode** once UI
ships).

---

## PRD requirements (new)

| ID | Requirement |
|----|-------------|
| REQ-SLV-01 | Workspaces group teams within one deployment; every team belongs to exactly one workspace. |
| REQ-SLV-02 | Teams may have support tier L1, L2, L3, NOC, or unset. |
| REQ-SLV-03 | Escalation paths define allowed tier-adjacent targets; API rejects others. |
| REQ-SLV-04 | Incident list/detail/ack/resolve/handoff/bounce use the API; timeline persists across refresh. |
| REQ-SLV-05 | Escalation picker shows only configured targets for the incident's owning team. |
| REQ-SLV-06 | Teams UI displays workspace and support tier; admin can configure escalation paths. |
| REQ-SLV-07 | Setup wizard includes workspace and escalation setup step. |
| REQ-SLV-08 | L1 and NOC tiers participate in escalation chain with validated paths (NOC→L1/L2, L1→L2, L2→L3). |
| REQ-SLV-09 | Integrations may be scoped to a workspace; Jira ticket create/update uses workspace `project_key` when set. |
| REQ-SLV-10 | Routing rules manageable in UI per workspace (label matchers, priority, target team). |
| REQ-SLV-11 | Timeline visibility unchanged (REQ-L2L3-03): all tiers see identical events; no role filtering. |

Existing REQ-L2L3-* remain in force for handoff behaviour, bounce, and analytics.

---

## Open questions

1. **Strict vs permissive escalation during migration** — Allow any valid-tier team until paths exist,
   or require configuration before escalate button enables?
2. **Default target when multiple paths exist** — Mark one path primary for one-click escalate without picker?
3. **Incident list default scope** — All workspaces for admins, or filter to teams the user belongs to?
4. **NOC as owning team** — Should routing default to NOC for unclassified alerts, or straight to L2?

Resolve in story planning; document decisions in PR descriptions.

---

## References

- Epic: [EPIC-11 Support levels & workspaces](../../backlog/epics/EPIC-11-support-levels-workspaces.md)
- Prior art: [EPIC-06 L2↔L3](../../backlog/epics/EPIC-06-l2-l3.md), [EPIC-09 Teams setup](../../backlog/epics/EPIC-09-teams-users-shifts.md)
- Data model: [03-data-model.md](../03-data-model.md)
- API: [04-api-spec.md](../04-api-spec.md)
- PRD: [01-prd.md](../01-prd.md) §4 (L2↔L3), new §7 (support levels — add when implemented)
