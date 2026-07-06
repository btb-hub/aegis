# EPIC-12 — Workspace admin (create & bind teams)

**Phase:** 11 (post-MVP UX)  
**Exit:** Admins create and edit workspaces from a dedicated page, assign existing teams to a
workspace without the setup wizard, and review routing/escalation impact before moving teams.

## Problem

Phase 10 shipped workspaces as a **backend model** and wired project scope through teams,
routing rules, and integrations ([EPIC-11](./EPIC-11-support-levels-workspaces.md)). The **only**
first-class UI for creating a workspace is the **Setup wizard** step that also creates L2/L3 teams
and an escalation path in one shot ([AEG-084](./EPIC-11-support-levels-workspaces.md)).

That flow breaks down for real admins:

1. **Teams often exist first.** After Phase 8, admins create teams on **Teams** before they think
   about project grouping. Those teams land in the migration **Default** workspace. Splitting into
   Platform / Data / Payments means re-homing teams — not creating greenfield L2/L3 pairs.
2. **No standalone workspace CRUD in the UI.** `POST /api/v1/workspaces` exists but is unreachable
   except via wizard or curl. **Workspace detail** (`/workspaces/{id}`) only manages routing rules;
   there is no edit name/slug, no team list, no “add existing team”.
3. **Teams cannot change workspace after creation.** `PATCH /api/v1/teams/{id}` ignores
   `workspace_id`; the Teams edit form does not expose workspace. Admins are stuck with whatever
   workspace was chosen at create time (usually Default).
4. **Wizard coupling feels wrong.** Workspace creation is bundled with escalation structure. An admin
   who only wants a project bucket before moving teams must fake L2/L3 names or use the API.

**User ask:** decouple workspace lifecycle from the wizard; support **create workspace → bind existing
teams**.

---

## Brainstorm — UX directions

### Option A — Dedicated **Workspaces** nav item (recommended)

Add **Workspaces** to the sidebar (between Teams and Incidents, or under Teams as a sibling).

| Screen | Purpose |
|--------|---------|
| **Workspaces list** | All workspaces; columns: name, slug, team count, routing rule count; **Create workspace** |
| **Workspace detail** | Tabs or sections: **Overview** (edit name/description/slug), **Teams** (in-workspace list + attach/detach), **Routing rules** (existing page content), **Integrations** (link or embed workspace-scoped rows) |

**Attach existing team flow:**

- On workspace detail → **Teams** → **Add existing team** opens a modal.
- Modal lists teams **not** in this workspace (default: all teams in Default workspace; filter/search by name).
- Multi-select → **Move to workspace** (bulk supported).
- Show warning if team has outgoing escalation paths pointing to teams in another workspace.

**Pros:** Discoverable; matches mental model (“I manage projects, then assign teams”).  
**Cons:** One more nav item; need empty states and breadcrumbs.

---

### Option B — Workspaces as a **Teams** sub-view (lighter nav)

Keep one **Teams** nav item. Add a workspace switcher at the top:

```text
[ All workspaces ▾ ]  [ Manage workspaces… ]
Teams table (filtered)
```

**Manage workspaces** opens a drawer or `/teams/workspaces` sub-route with list + create.

**Pros:** No new top-level nav; teams and workspaces stay co-located.  
**Cons:** Easy to miss; “create workspace” is buried; less room for routing rules per workspace.

---

### Option C — **Inline create** only (minimal)

Add **Create workspace** on Teams page (next to workspace filter) and allow workspace change in team
edit modal. No list page — workspace detail stays deep-linked from team rows.

**Pros:** Smallest diff.  
**Cons:** No overview of all projects; bulk move awkward; does not fix “hub” for routing rules.

---

### Option D — **Wizard becomes optional shortcut**

Keep wizard for greenfield installs but add Option A (or B) for day-2 admin. Wizard step 3 becomes:
“Quick start: create workspace + L2/L3 pair” with **Skip — I'll configure in Workspaces** link.

**Pros:** Best of both worlds for NFR-1 first-day setup vs ongoing admin.  
**Cons:** Two paths to document; wizard step must not feel mandatory.

---

### Recommendation (locked)

Ship **Option A + D** with these decisions confirmed:

| Decision | Choice | Rationale |
|----------|--------|-----------|
| **Nav** | Top-level **Workspaces** in sidebar (between Teams and Incidents) | Project scope is a first-class concept; burying under Teams caused the wizard-only gap |
| **Bulk move API** | `POST /api/v1/workspaces/{id}/teams` with `{ "team_ids": [...] }` | Single transaction: validate all teams, check all path conflicts, commit all or none. Repeated PATCH leaves half-moved state on partial failure |
| **Conflict handling (v1)** | **Block only** — no `force` flag, no auto-delete paths | Surprises admins less; paths are intentional config. UI links to team path admin to fix, then retry |

Single-team moves still use `PATCH /api/v1/teams/{id}` with `workspace_id` (same validation rules).
Bulk assign endpoint reuses the same validation helper.

---

## Brainstorm — binding & data integrity

Moving a team between workspaces touches more than `teams.workspace_id`.

| Concern | Behaviour to decide |
|---------|---------------------|
| **Escalation paths** | **Block** move until admin removes conflicting paths. Return `409` with path IDs in `details`. No silent deletion in v1. |
| **Cross-workspace paths** | Same block rule. `cross_workspace=true` paths remain valid only when both endpoint teams still satisfy path rules after the move. Admin removes or reconfigures manually. |
| **Routing rules** | Rules reference `team_id` only. They keep working after move; workspace detail filter may show them under the new workspace. No migration needed. |
| **Open incidents** | Incidents reference `team_id`. No change on move. Historical correctness OK. |
| **Workspace integrations** | Scoped by `workspace_id`. Moving team changes which Jira project key applies to **new** tickets for that team's incidents. Document in UI. |
| **Default workspace** | Keep non-deletable (or deletable only when empty). Prevents orphan teams. |
| **Delete workspace** | Reject when any team, escalation path, or workspace-scoped integration remains. Return `409` with counts. |

---

## Brainstorm — API gaps (today)

| Gap | Fix |
|-----|-----|
| `PATCH /teams/{id}` ignores `workspace_id` | Accept optional `workspace_id`; validate workspace exists; run path conflict check |
| No bulk assign | `POST /workspaces/{id}/teams` body `{ "team_ids": ["uuid", ...] }` — atomic, all-or-nothing |
| List workspaces has no counts | Extend `GET /workspaces` items with `team_count`, `routing_rule_count` (computed) |
| Delete workspace unsafe | Service-layer guard before DELETE |

---

## Solution (four tracks)

1. **Workspaces hub UI** — list, create, edit workspace metadata.
2. **Bind existing teams** — team edit workspace selector + workspace detail “add teams” with bulk move.
3. **API & validation** — PATCH team workspace (single), `POST /workspaces/{id}/teams` (bulk), block-only path guards.
4. **Wizard decoupling** — setup step 3 optional; point to Workspaces; keep “quick L2/L3 structure” as shortcut.

---

### AEG-088 — Workspaces list page and create/edit UI

- **Status:** In Review
- **Depends on:** AEG-079 (workspaces API — Done on main)
- **PRD:** REQ-SLV-01, REQ-SLV-07 (setup ergonomics)
- **Acceptance:**
  - [ ] New route `/workspaces` — list all workspaces (admin: create button; all roles: read list)
  - [ ] Columns: name, slug, team count, description (truncated)
  - [ ] **Create workspace** modal: name (required), description (optional); slug auto-generated, editable on edit
  - [ ] Row links to `/workspaces/{id}` (existing detail)
  - [ ] **Edit workspace** on detail or modal: name, slug, description via `PATCH /api/v1/workspaces/{id}`
  - [ ] Nav: top-level **Workspaces** in sidebar (between Teams and Incidents)
  - [ ] Locale strings `en` + `ru`; Storybook story if new shared component; web tests for list + create
  - [ ] Update [`docs/user-guide.md`](../docs/user-guide.md) — replace wizard-only create flow

**Plan:** `WorkspacesPage.tsx`; extend `workspacesApi.ts` with `updateWorkspace`. Branch: `feat/workspaces-AEG-088-list-ui`.

---

### AEG-089 — Team workspace reassignment (API)

- **Status:** In Review
- **Depends on:** AEG-079
- **PRD:** REQ-SLV-01
- **Acceptance:**
  - [ ] `PATCH /api/v1/teams/{id}` accepts optional `workspace_id` (single-team move)
  - [ ] `POST /api/v1/workspaces/{id}/teams` body `{ "team_ids": ["uuid", ...] }` (bulk move, admin)
  - [ ] Bulk endpoint is **atomic**: validate all teams exist, none already in target workspace,
        run path conflict check for each; commit all updates in one transaction or return error with
        no partial moves
  - [ ] Validates workspace exists; rejects unknown UUIDs
  - [ ] If any escalation path (incoming or outgoing) would become invalid, return `409` with
        `{ code, message, details: { blocked_teams: [{ team_id, paths: [...] }] } }`
  - [ ] **No `force` flag in v1** — admin removes paths on team detail, then retries
  - [ ] Cross-workspace paths: document validation rule in API spec
  - [ ] Unit tests: single PATCH happy/blocked; bulk all succeed; bulk one blocked → none moved
  - [ ] Update [`docs/04-api-spec.md`](../docs/04-api-spec.md)

**Plan:** `TeamService` move helper + `EscalationService.ValidateTeamWorkspaceMove`; bulk handler on
`WorkspaceHandler`. Branch: `feat/teams-AEG-089-workspace-move`.

---

### AEG-090 — Bind teams in UI (edit + workspace detail)

- **Status:** In Review
- **Depends on:** AEG-088, AEG-089
- **PRD:** REQ-SLV-06
- **Acceptance:**
  - [ ] **Teams** edit modal: **Workspace** dropdown; saves via `PATCH` with `workspace_id`
  - [ ] **Workspace detail** → **Teams** section: table of teams in this workspace
  - [ ] **Add existing teams** button (admin): modal with searchable multi-select; calls
        `POST /api/v1/workspaces/{id}/teams` (not repeated PATCH)
  - [ ] On `409`: show per-team path conflicts from `details.blocked_teams`; link to team detail
  - [ ] Empty state: “No teams in this workspace yet — create one or add existing”
  - [ ] Web tests: change workspace on edit; bulk add from detail

**Plan:** Extend `TeamsPage` form + `WorkspaceDetailPage` teams section. Branch: `feat/workspaces-AEG-090-bind-teams-ui`.

---

### AEG-091 — Workspace list enrichment and safe delete

- **Status:** In Review
- **Depends on:** AEG-088
- **PRD:** REQ-SLV-01
- **Acceptance:**
  - [ ] `GET /workspaces` returns `team_count` and `routing_rule_count` per item
  - [ ] `DELETE /workspaces/{id}` rejected with `409` when teams, paths, or workspace integrations exist
        (details include counts)
  - [ ] **Default** workspace (`00000000-0000-0000-0000-000000000001`) not deletable — `403` or `409`
  - [ ] Workspaces list UI: delete with confirm (admin); show error toast with reason
  - [ ] Tests for service guards and handler responses

**Plan:** SQL aggregates in `ListWorkspaces`; delete guard in `WorkspaceService`. Branch: `feat/workspaces-AEG-091-safe-delete`.

---

### AEG-092 — Decouple setup wizard from workspace creation

- **Status:** In Review
- **Depends on:** AEG-088, AEG-090
- **PRD:** REQ-SLV-07, NFR-1
- **Acceptance:**
  - [ ] Wizard step 3 (workspace): primary CTA **Open workspaces** linking to `/workspaces`; secondary
        **Quick setup** retains create workspace + L2/L3 + path in one click (current behaviour)
  - [ ] Copy clarifies: “Already have teams? Create a workspace and assign them under Workspaces.”
  - [ ] Wizard skip logic: if ≥1 non-default workspace exists with ≥1 team, mark step skippable
  - [ ] Update [`docs/07-setup-deployment.md`](../docs/07-setup-deployment.md) and user guide
  - [ ] Setup wizard tests updated

**Plan:** Copy + link changes in `SetupWizardPage.tsx`; no removal of quick path. Branch: `feat/setup-AEG-092-wizard-decouple`.

---

## Dependency graph

```
AEG-079 (workspaces API — Done)
  ├── AEG-088 (workspaces list UI)
  │     └── AEG-091 (counts + safe delete)
  ├── AEG-089 (PATCH team workspace_id)
  │     └── AEG-090 (bind teams UI)
  └── AEG-088 + AEG-090 → AEG-092 (wizard decouple)
```

**Suggested order:** AEG-088 → AEG-089 → AEG-090 (parallel AEG-091) → AEG-092.

---

## Out of scope

- Drag-and-drop team ordering between workspaces
- Automatic migration wizard (“split Default into N workspaces by team name prefix”)
- Member/viewer ability to move teams (admin only)
- Changing workspace on open incidents retroactively
- Multi-tenant / org-level isolation
- **`force` move** — auto-delete conflicting escalation paths and move in one action (future story if needed)
- Workspaces nested under Teams nav (rejected — top-level only)

---

## Definition of done (epic)

- [ ] Admin creates a workspace from **Workspaces** without opening Setup wizard
- [ ] Admin moves existing teams from Default (or any workspace) to the new workspace
- [ ] Blocked moves explain escalation path conflicts clearly
- [ ] Setup wizard documents and links to Workspaces; quick L2/L3 path remains for greenfield
- [ ] `make lint type test` green; API spec and user guide updated

---

## Decisions (locked)

| # | Question | Decision |
|---|----------|----------|
| 1 | Nav placement | **Top-level Workspaces** in sidebar (between Teams and Incidents) |
| 2 | Bulk move | **`POST /workspaces/{id}/teams`** — atomic, all-or-nothing; single-team via `PATCH /teams/{id}` |
| 3 | Path conflicts | **Block only in v1** — no `force` flag; admin fixes paths then retries |
