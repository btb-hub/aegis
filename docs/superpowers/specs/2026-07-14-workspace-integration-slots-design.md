# Workspace integration slots — design

**Date:** 2026-07-14  
**Status:** Approved for planning (product decisions captured below)  
**Related:** EPIC-11 AEG-085, EPIC-12, EPIC-13; `docs/features/support-levels-and-workspaces.md`

## Problem

Admins can create integrations, but the link to **workspaces → alerts → incidents** is unclear. Today:

- At most one **global** row per kind (`jira` / `slack` / `express`).
- Optional **workspace** rows exist mainly as thin Jira `project_key` overrides.
- Runtime resolves workspace via **team**, then merges overrides only on the **new-incident** path; escalate/handoff still use **global-only** registries.
- UI can invent “extra” rows but does not show a stable, always-visible binding per workspace.

Operators need: each workspace has clear connector slots; hybrid inherit vs custom credentials; fallback to global; incidents still work in Aegis when connectors are missing, with an explicit notice.

## Goals

1. Every workspace always has **three connector slots** (Jira, Slack, eXpress), visible and configurable.
2. Each slot is either **Inherit** (global credentials + optional overlays) or **Custom** (full workspace credentials). Partial secret mixing is forbidden.
3. Resolution order is identical for new incident, escalate, and handoff.
4. Missing connector → **soft skip** that provider; incident lifecycle in Aegis continues; non-blocking notice.
5. Admin UX: Workspace page Integrations **and** `/integrations` inventory, with inheritance obvious on create and when observing later.

## Non-goals

- New providers (Mattermost, Telegram, …).
- Multiple Jira (or Slack/eXpress) instances **per workspace**.
- Changing OIDC / login credentials.
- Requiring integrations to create or acknowledge incidents.
- Backfilling tickets/pages for past incidents when a slot is configured later.
- Denormalizing `incidents.workspace_id` (optional follow-up for filters only).

## Decisions (from brainstorming)

| Topic | Choice |
|-------|--------|
| Credential model | Hybrid: workspace may use full own credentials or inherit global |
| Missing workspace config | Fall back to global |
| Missing global too | Soft skip; Aegis-only ops; notice |
| Admin surfaces | Both Workspace Integrations + `/integrations` |
| Config completeness | Inherit (overlays only) **or** Custom (all required secrets); if any secret set → all required |
| Shape | Always-provisioned slots per workspace (Approach 3) |

## Architecture

### Units

1. **Global integration store** — existing `integrations` rows with `workspace_id IS NULL` (≤1 per kind). Source of credentials for Inherit mode.
2. **Workspace slots** — always three rows per workspace `(workspace_id, kind ∈ {jira,slack,express})`, created on workspace create and backfilled for existing workspaces.
3. **Slot mode** — first-class field on the workspace row: `inherit` | `custom` (name TBD in implementation; must be stored, not only inferred).
4. **Connector resolver** — single service/function used by worker (alert notify, escalate, handoff) and by Test connection: inputs `(workspaceID, kind)` → provider config or “unavailable” reason.
5. **Soft-skip notifier** — writes a timeline (and consistent API/UI message) when a kind is skipped.
6. **Admin UI** — Workspace Integrations section + `/integrations` list/filter/configure using the same APIs.

### Data model

Extend workspace-scoped `integrations` rows (keep uniqueness `(workspace_id, kind)`):

- `mode` — `inherit` | `custom` (required for workspace rows; globals ignore / null).
- `enabled` — unchanged.
- `config` JSON:
  - **Inherit:** only non-secret overlays (`project_key` for Jira; optional Slack channel id if product keeps it). No secret keys stored (or rejected on write).
  - **Custom:** full required fields for that kind (same rules as EPIC-13 global validation).

**Provisioning:**

- `CreateWorkspace` → insert three disabled-or-enabled Inherit slots (product default: `enabled=true`, `mode=inherit`, empty overlays).
- Migration: for each existing workspace, ensure three slots; map current workspace Jira override rows to `mode=inherit` with `project_key`; create missing slack/express Inherit slots.

**Globals** remain optional. Inherit with no global → resolver returns unavailable for that kind.

### Resolution algorithm

```
resolve(workspaceID, kind):
  slot = GetWorkspaceSlot(workspaceID, kind)   // after backfill, always exists
  if slot is nil or not slot.enabled:
    return unavailable(reason: slot_disabled_or_missing)
  if slot.mode == custom:
    if configComplete(custom): return slot.config
    return unavailable(reason: custom_incomplete)
  // inherit
  global = GetGlobal(kind)
  if global is nil or not global.enabled:
    return unavailable(reason: no_global)
  return merge(global.config, slot.overlays)
```

Unavailable → caller skips that provider and records a notice; other kinds still resolve independently.

### Runtime data flow

```
Alert webhook
  → process_alert
  → routing → team_id
  → incident (team_id)
  → workspaceID = team.workspace_id
  → for each kind: resolve(workspaceID, kind)
       → CreateTicket / SendPage or soft-skip + timeline
Escalate / handoff_notify
  → same resolve(workspaceID, kind)  // not global-only list
```

Workspace is still derived from the incident’s team. No requirement to stamp `incidents.workspace_id` in this design.

### API / validation (sketch)

- Keep list/get with secret redaction (`***`) from EPIC-13.
- PATCH workspace slot: `mode`, `enabled`, `config`.
  - Mode → Inherit: strip secrets from stored config; allow overlays only.
  - Mode → Custom: require full credentials before enable/test success; blank secrets on PATCH keep existing (EPIC-13 rule).
- Test connection: run `resolve` then provider `TestConnection`; surface incomplete inherit/custom clearly.
- Creating a “workspace integration” of a new kind is unnecessary once slots exist; UI configures slots instead of POSTing new kinds for that workspace.

### UI / UX

**Workspace detail → Integrations**

- Always three rows: Jira, Slack, eXpress.
- Badges: mode (Inherit / Custom); status (Ready / Needs setup / Using global / Missing — no global / Disabled).
- Configure: mode control first; copy explains inheritance; Custom requires all secrets; never show live secrets.
- Enable / Disable / Test.

**`/integrations`**

- Inventory of globals + all workspace slots.
- Filters: Global | Workspace, kind, status.
- Configure uses the same dialog; “Add” remains for missing **global** kinds.

**Absence notice**

- Timeline event when ticket/page skipped (kind + reason + where to fix: workspace Integrations or global).
- Same wording if a user-facing toast already exists for that failure path.

### Error handling

| Situation | Behaviour |
|-----------|-----------|
| Custom incomplete | Validation on save/test; runtime soft-skip if somehow enabled incomplete |
| Inherit, no global | Soft-skip + notice “No global {kind}; add global or switch slot to Custom” |
| Slot disabled | Soft-skip + notice |
| Provider HTTP failure | Existing validation/error messages; do not roll back incident |

### Testing

- Resolver unit tests: custom / inherit+overlay / inherit no global / disabled / soft-skip reason codes.
- Worker: escalate and handoff use resolver (regression vs global-only).
- API: mode transitions strip or require secrets correctly; redaction unchanged.
- Web: three slots always rendered; Inherit vs Custom form fields; incomplete Custom blocks save; status badges.

### Suggested delivery slices (for a later plan / epic)

1. Schema `mode` + backfill/provision slots; resolver + worker wiring (alert/escalate/handoff) + timeline soft-skip.
2. API validation for Inherit vs Custom + Test via resolver.
3. Workspace Integrations UI.
4. `/integrations` inventory filters + align create flows with slots.

## Open product defaults (fixed here to avoid ambiguity)

- New slots default: `mode=inherit`, `enabled=true`, empty overlays.
- Switch Custom → Inherit: confirm, then delete workspace secrets.
- Switch Inherit → Custom: start with empty secrets (no silent copy-from-global). Optional “Copy from global” can be a later enhancement.
- Slack Inherit overlay: include `channel_id` only if paging already supports channel targeting; otherwise Inherit has no overlay fields until that exists. Jira Inherit always supports `project_key`.

## References

- Current merge helper: `pkg/db/routing.go` (`ListEnabledIntegrationsForWorkspace`, `mergeIntegrationConfig`)
- Gaps to close: `apps/worker/internal/processor/escalate.go`, `handoff_notify.go` (global-only lists)
- Credential admin: EPIC-13 / `/integrations`
