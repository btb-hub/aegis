# Workspace IntegrationSlots Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Give every workspace always-on Jira/Slack/eXpress slots with Inherit vs Custom credentials, resolve connectors the same way for new incidents / escalate / handoff, and soft-skip missing providers with a timeline notice.

**Architecture:** Persist `mode` on workspace `integrations` rows; provision three slots per workspace; put a pure resolver in `pkg/integrations/resolve` used by the worker and Test API; admin configures slots on Workspace detail and `/integrations`.

**Tech Stack:** Go 1.22+, PostgreSQL 16, golang-migrate, Gin, React + TypeScript + Vitest, react-i18next.

**Spec:** [`docs/superpowers/specs/2026-07-14-workspace-integration-slots-design.md`](../specs/2026-07-14-workspace-integration-slots-design.md)

## Global Constraints

- One global row per kind (`workspace_id IS NULL`); one slot per `(workspace_id, kind)`.
- Workspace slot `mode` is `inherit` | `custom` (required when `workspace_id` set; NULL for globals).
- New slots default: `mode=inherit`, `enabled=true`, empty overlays.
- Inherit: no secret keys in stored config; Jira overlay may include `project_key` only (no Slack `channel_id` until paging supports it).
- Custom: full credentials per EPIC-13 / provider `NewFromJSON`; if any secret is sent, all required secrets for that kind are required.
- Soft-skip unavailable providers; never fail alert ingest solely for missing connectors.
- User-facing strings: en + ru. Secrets redacted as `***` in API responses.
- `make lint type test` green; business-logic coverage ≥ 90%.
- One story-sized PR branch preferred; commits Conventional Commits.

---

## File map

| File | Responsibility |
|------|----------------|
| `db/migrations/000016_integration_slot_mode.up.sql` / `.down.sql` | Add `mode`; backfill three slots per workspace |
| `pkg/db/models.go` | `Integration.Mode *string` |
| `pkg/db/routing.go` | Scan/write `mode`; `EnsureWorkspaceSlots`; get slot/global helpers |
| `pkg/db/workspaces.go` | Call `EnsureWorkspaceSlots` after create (or from service layer) |
| `pkg/integrations/registry.go` | Extend `IntegrationRow` with `Mode` and `WorkspaceID` if needed by loader |
| `pkg/integrations/resolve/resolve.go` | Pure `Resolve(kind, slot, global)` + reason codes |
| `pkg/integrations/resolve/resolve_test.go` | Resolver unit tests |
| `apps/api/internal/service/integration.go` | Validation Inherit/Custom; status for JSON; Test via resolve; strip secrets on Inherit |
| `apps/api/internal/service/workspace.go` | Provision slots on create |
| `apps/worker/internal/processor/alert.go` | Build registry via resolve; timeline soft-skip |
| `apps/worker/internal/processor/escalate.go` | Workspace-aware resolve |
| `apps/worker/internal/processor/handoff_notify.go` | Workspace-aware resolve |
| `apps/web/.../WorkspaceDetailPage.tsx` | Integrations section (3 slots) |
| `apps/web/.../IntegrationsPage.tsx` | Inventory filters; stop creating ad-hoc workspace kinds |
| `apps/web/src/locales/en/common.json` + `ru` | Copy for mode/status/notices |
| `docs/03-data-model.md`, `docs/04-api-spec.md`, `docs/integrations/README.md` | Document mode + resolution |

---

### Task 1: Migration — `mode` column + backfill slots

**Files:**
- Create: `db/migrations/000016_integration_slot_mode.up.sql`
- Create: `db/migrations/000016_integration_slot_mode.down.sql`
- Modify: `pkg/db/models.go` (`Integration` struct)

**Interfaces:**
- Produces: DB column `integrations.mode` (`text`, nullable); check `mode IS NULL OR mode IN ('inherit','custom')`; workspace rows after migration always have `mode` set.

- [ ] **Step 1: Write the up migration**

```sql
-- 000016_integration_slot_mode.up.sql
ALTER TABLE integrations
  ADD COLUMN mode TEXT;

ALTER TABLE integrations
  ADD CONSTRAINT integrations_mode_check
  CHECK (mode IS NULL OR mode IN ('inherit', 'custom'));

-- Existing workspace rows become inherit (typical project_key overrides).
UPDATE integrations
SET mode = 'inherit'
WHERE workspace_id IS NOT NULL AND mode IS NULL;

-- Ensure three slots per workspace.
INSERT INTO integrations (kind, name, config, enabled, workspace_id, mode)
SELECT k.kind, k.kind, '{}'::jsonb, true, w.id, 'inherit'
FROM workspaces w
CROSS JOIN (VALUES ('jira'), ('slack'), ('express')) AS k(kind)
WHERE NOT EXISTS (
  SELECT 1 FROM integrations i
  WHERE i.workspace_id = w.id AND i.kind = k.kind
);
```

- [ ] **Step 2: Write the down migration**

```sql
-- 000016_integration_slot_mode.down.sql
ALTER TABLE integrations DROP CONSTRAINT IF EXISTS integrations_mode_check;
ALTER TABLE integrations DROP COLUMN IF EXISTS mode;
-- Do not delete auto-created slots on down (data-preserving); only drop mode.
```

- [ ] **Step 3: Extend the Go model**

In `pkg/db/models.go` on `Integration`:

```go
Mode *string `json:"mode,omitempty"` // "inherit" | "custom" for workspace rows; nil for global
```

- [ ] **Step 4: Commit**

```bash
git add db/migrations/000016_integration_slot_mode.up.sql db/migrations/000016_integration_slot_mode.down.sql pkg/db/models.go
git commit -m "feat: add integration slot mode migration and model field"
```

---

### Task 2: Store helpers — scan mode, EnsureWorkspaceSlots, get slot/global

**Files:**
- Modify: `pkg/db/routing.go` (all integration SQL scan/insert/update that touch integrations)
- Test: `pkg/db` integration tests if present; otherwise unit-style tests via service later — prefer adding `EnsureWorkspaceSlots` covered by Task 3 resolver fixtures + workspace create tests

**Interfaces:**
- Produces:
  - `func (s *Store) EnsureWorkspaceSlots(ctx context.Context, workspaceID uuid.UUID) error`
  - `func (s *Store) GetIntegrationByKind(ctx, kind) (Integration, error)` — already exists (global)
  - `func (s *Store) GetWorkspaceIntegration(ctx, workspaceID uuid.UUID, kind string) (Integration, error)`
  - All read/write methods populate `Mode`

- [ ] **Step 1: Update SELECT lists** in `ListIntegrations`, `GetIntegration`, `GetIntegrationByKind`, `UpsertIntegration`, `UpdateIntegration`, `ListEnabledIntegrations*` to include `mode` and scan into `item.Mode`.

- [ ] **Step 2: Add GetWorkspaceIntegration**

```go
func (s *Store) GetWorkspaceIntegration(ctx context.Context, workspaceID uuid.UUID, kind string) (Integration, error) {
	const q = `
SELECT id, kind, name, config, enabled, workspace_id, mode, created_at, updated_at
FROM integrations
WHERE workspace_id = $1 AND kind = $2`
	// scan Mode into *string / pgtype as needed
}
```

- [ ] **Step 3: Add EnsureWorkspaceSlots**

```go
func (s *Store) EnsureWorkspaceSlots(ctx context.Context, workspaceID uuid.UUID) error {
	const q = `
INSERT INTO integrations (kind, name, config, enabled, workspace_id, mode)
VALUES ($1, $1, '{}'::jsonb, true, $2, 'inherit')
ON CONFLICT DO NOTHING`
	// Note: unique index is (workspace_id, kind) — use ON CONFLICT ON CONSTRAINT or
	// INSERT ... WHERE NOT EXISTS loop for jira/slack/express.
	for _, kind := range []string{"jira", "slack", "express"} {
		// insert if missing
	}
	return nil
}
```

Use `WHERE NOT EXISTS` (same as migration) if `ON CONFLICT` target is awkward with partial unique indexes.

- [ ] **Step 4: Update Upsert/Update** to accept and persist `mode` for workspace rows (`UpdateIntegration` signature grows `mode *string` or pass via a params struct — prefer:

```go
func (s *Store) UpdateIntegration(ctx context.Context, id uuid.UUID, name string, config json.RawMessage, enabled bool, mode *string) (Integration, error)
```

Update all call sites/mocks in the same commit.

- [ ] **Step 5: Commit**

```bash
git add pkg/db/routing.go pkg/db/models.go apps/api/**/*_test.go
git commit -m "feat: persist integration mode and ensure workspace slots"
```

---

### Task 3: Pure connector resolver

**Files:**
- Create: `pkg/integrations/resolve/resolve.go`
- Create: `pkg/integrations/resolve/resolve_test.go`

**Interfaces:**
- Produces:

```go
package resolve

const (
	ReasonOK               = ""
	ReasonSlotDisabled     = "slot_disabled"
	ReasonSlotMissing      = "slot_missing"
	ReasonCustomIncomplete = "custom_incomplete"
	ReasonNoGlobal         = "no_global"
	ReasonGlobalDisabled   = "global_disabled"
)

type Input struct {
	Kind    string
	Slot    *Slot // nil if no workspace row
	Global  *Slot // nil if no global row
}

type Slot struct {
	Mode    string // inherit|custom|"" for global
	Enabled bool
	Config  []byte
}

type Result struct {
	OK     bool
	Config []byte
	Reason string
}

func Resolve(in Input) Result
func MergeConfig(global, overlay []byte) ([]byte, error) // reuse logic from db.mergeIntegrationConfig — move shared merge here or call duplicate carefully
```

- [ ] **Step 1: Write failing tests**

```go
func TestResolveCustomComplete(t *testing.T) { /* mode custom + complete jira JSON → OK */ }
func TestResolveCustomIncomplete(t *testing.T) { /* → ReasonCustomIncomplete */ }
func TestResolveInheritMergesProjectKey(t *testing.T) { /* global creds + slot project_key */ }
func TestResolveInheritNoGlobal(t *testing.T) { /* → ReasonNoGlobal */ }
func TestResolveSlotDisabled(t *testing.T) { /* → ReasonSlotDisabled */ }
func TestResolveNilSlot(t *testing.T) { /* → ReasonSlotMissing */ }
```

For “complete”, call the same field checks as providers (trim empty `base_url`/`email`/`api_token`/`project_key` for jira; etc.) — either invoke `jira.NewFromJSON` / `slack.NewFromJSON` / `express.NewFromJSON` or duplicate the emptiness checks in `resolve.ConfigComplete(kind, raw)`.

- [ ] **Step 2: Run tests — expect FAIL**

```bash
cd /path/to/repo && go test ./pkg/integrations/resolve/ -count=1
```

Expected: package not found or missing `Resolve`.

- [ ] **Step 3: Implement `Resolve` + `ConfigComplete`**

```go
func Resolve(in Input) Result {
	if in.Slot == nil {
		return Result{Reason: ReasonSlotMissing}
	}
	if !in.Slot.Enabled {
		return Result{Reason: ReasonSlotDisabled}
	}
	switch in.Slot.Mode {
	case "custom":
		if !ConfigComplete(in.Kind, in.Slot.Config) {
			return Result{Reason: ReasonCustomIncomplete}
		}
		return Result{OK: true, Config: in.Slot.Config}
	default: // inherit
		if in.Global == nil {
			return Result{Reason: ReasonNoGlobal}
		}
		if !in.Global.Enabled {
			return Result{Reason: ReasonGlobalDisabled}
		}
		merged, err := MergeConfig(in.Global.Config, in.Slot.Config)
		if err != nil || !ConfigComplete(in.Kind, merged) {
			return Result{Reason: ReasonCustomIncomplete} // or ReasonNoGlobal if incomplete due to global
		}
		return Result{OK: true, Config: merged}
	}
}
```

- [ ] **Step 4: Run tests — expect PASS**

```bash
go test ./pkg/integrations/resolve/ -count=1
```

- [ ] **Step 5: Commit**

```bash
git add pkg/integrations/resolve/
git commit -m "feat: add workspace connector resolve package"
```

---

### Task 4: Soft-skip timeline + wire AlertProcessor

**Files:**
- Modify: `apps/worker/internal/processor/alert.go`
- Modify: `apps/worker/internal/processor/alert_test.go` / `alert_flow_test.go` as needed
- Optionally create: `apps/worker/internal/processor/connector_skip.go` with shared notice helper

**Interfaces:**
- Consumes: `resolve.Resolve`, `GetTeamWorkspaceID`, `GetWorkspaceIntegration`, `GetIntegrationByKind`
- Produces: timeline kind `integration_skipped` with payload `{"kind":"jira","reason":"no_global","message":"..."}`

- [ ] **Step 1: Write failing test** — registry load uses workspace slot; when jira unavailable, `AppendTimelineEvent` called with `integration_skipped` and incident still created.

- [ ] **Step 2: Run test — expect FAIL**

- [ ] **Step 3: Replace `loadRegistry` to resolve each kind**

```go
func (p *AlertProcessor) loadRegistry(ctx context.Context, teamID uuid.UUID) (*integrations.Registry, []skipNotice, error) {
	workspaceID, err := p.store.GetTeamWorkspaceID(ctx, teamID)
	// for kind in jira,slack,express:
	//   slot,_ := GetWorkspaceIntegration
	//   global,_ := GetIntegrationByKind (ignore ErrNoRows)
	//   res := resolve.Resolve(...)
	//   if res.OK { rows = append(rows, IntegrationRow{Kind, Config: res.Config, Enabled: true}) }
	//   else { notices = append(...) }
	loader.RegisterFromRows(reg, rows, publicURL)
	return reg, notices, nil
}
```

Extend the processor store interface with `GetWorkspaceIntegration` / keep using existing methods.

- [ ] **Step 4: In `notifyIntegrations`, after loadRegistry, for each notice append timeline** before/after ForEachTicket/Chat:

```go
_ = p.store.AppendTimelineEvent(ctx, incident.ID, "integration_skipped", nil, mustJSON(map[string]string{
  "kind": n.Kind, "reason": n.Reason, "message": skipMessage(n),
}))
```

English source message strings, e.g. `Jira skipped: no global connector. Configure global Jira or set the workspace slot to Custom.`

- [ ] **Step 5: Run worker tests — expect PASS**

```bash
go test ./apps/worker/internal/processor/ -count=1
```

- [ ] **Step 6: Commit**

```bash
git add apps/worker/internal/processor/
git commit -m "feat: resolve workspace connectors on alert notify with soft-skip"
```

---

### Task 5: Wire escalate + handoff_notify to the same resolver

**Files:**
- Modify: `apps/worker/internal/processor/escalate.go`
- Modify: `apps/worker/internal/processor/handoff_notify.go`
- Modify: their tests / mocks

**Interfaces:**
- Consumes: same resolve helpers as Task 4  
- Produces: escalate/handoff no longer call `ListEnabledIntegrations()` alone

- [ ] **Step 1: Write failing tests** proving escalate with only a workspace custom/inherit slot pages using that config, and does not page when slot inherit and global missing (timeline skip).

- [ ] **Step 2: Run — expect FAIL** (still global-only)

- [ ] **Step 3: Change store interfaces** on escalate/handoff to include `GetTeamWorkspaceID`, `GetWorkspaceIntegration`, `GetIntegrationByKind` (or a single `ListResolvedIntegrationRows(ctx, workspaceID)` on the store/service to DRY with alert). Prefer extracting:

```go
// apps/worker/internal/processor/registry.go
func loadWorkspaceRegistry(ctx, store, teamID, publicURL) (reg, notices, err)
```

shared by alert, escalate, handoff.

- [ ] **Step 4: Implement shared loader; switch both processors**

- [ ] **Step 5: Run**

```bash
go test ./apps/worker/internal/processor/ -count=1
```

- [ ] **Step 6: Commit**

```bash
git add apps/worker/internal/processor/
git commit -m "feat: use workspace connector resolve on escalate and handoff"
```

---

### Task 6: Provision slots on workspace create + API validation

**Files:**
- Modify: `apps/api/internal/service/workspace.go` (`Create`)
- Modify: `apps/api/internal/service/integration.go` (Upsert/Update/Test/JSON)
- Modify: `apps/api/internal/handler/integration.go` if PATCH body needs `mode`
- Modify: tests under `apps/api/internal/service/` and `handler/`

**Interfaces:**
- Produces: `IntegrationJSON` includes `mode`, `slot_status` (`ready` | `needs_setup` | `using_global` | `missing` | `disabled`)
- `Update` accepts `mode`; Inherit strips secrets; Custom validates full config

- [ ] **Step 1: Failing tests**

```go
func TestCreateWorkspaceProvisionsThreeSlots(t *testing.T) { ... }
func TestUpdateSlotInheritStripsSecrets(t *testing.T) { ... }
func TestUpdateSlotCustomRequiresAllSecrets(t *testing.T) { ... }
func TestTestConnectionUsesResolve(t *testing.T) { ... }
```

- [ ] **Step 2: Run — expect FAIL**

- [ ] **Step 3: In `WorkspaceService.Create`**, after `CreateWorkspace`, call `EnsureWorkspaceSlots`.

- [ ] **Step 4: Integration service rules**

```go
secretKeys := []string{"api_token", "bot_token", "signing_secret", "secret_key"}

// On workspace Update/Upsert:
if mode == "inherit" {
  config = stripKeys(config, secretKeys)
  // jira: allow project_key only (optional empty)
} else if mode == "custom" {
  if err := validateGlobalIntegrationConfig(kind, config, publicURL); err != nil { return err }
}
```

Disallow `POST /integrations` creating a new workspace kind if the slot already exists (return existing via PATCH) — or map POST workspace+kind to update-slot behavior. Prefer: UI only PATCHes slot ids; POST remains for **global** kinds only when none exists.

- [ ] **Step 5: IntegrationJSON**

```go
out["mode"] = mode
out["slot_status"] = computeSlotStatus(item, globalExists) // for workspace rows
out["config_complete"] = ... // keep EPIC-13 flag as useful for globals; for slots prefer slot_status
```

- [ ] **Step 6: Test()** loads slot + global, `resolve.Resolve`, then provider TestConnection; map reasons to validation messages.

- [ ] **Step 7: Run**

```bash
go test ./apps/api/internal/service/ ./apps/api/internal/handler/ -count=1
```

- [ ] **Step 8: Commit**

```bash
git add apps/api/internal/service/ apps/api/internal/handler/ pkg/db/
git commit -m "feat: provision slots and validate inherit vs custom modes"
```

---

### Task 7: Workspace detail Integrations UI

**Files:**
- Modify: `apps/web/src/pages/WorkspaceDetailPage.tsx`
- Modify: `apps/web/src/pages/WorkspaceDetailPage.test.tsx`
- Create (optional): `apps/web/src/components/integrations/WorkspaceSlotsPanel.tsx`
- Modify: `apps/web/src/locales/en/common.json`, `apps/web/src/locales/ru/common.json`
- Reuse: `IntegrationConfigFields.tsx` with mode prop

**Interfaces:**
- Consumes: `GET /integrations` (filter client-side by `workspace_id`) or future `GET /workspaces/:id/integrations` — **YAGNI:** filter list by `workspace_id` for this task.
- Produces: always three rows UI; Configure sets `mode` + config via PATCH

- [ ] **Step 1: Add locale keys** (`workspaces.integrations.*`: title, mode_inherit, mode_custom, status_*, configure, inherit_help, custom_help, switch_confirm)

- [ ] **Step 2: Failing Vitest** — workspace detail shows Jira/Slack/eXpress even when API returns three inherit slots; Configure with Inherit shows project_key only for jira.

- [ ] **Step 3: Run**

```bash
cd apps/web && npx vitest --run src/pages/WorkspaceDetailPage.test.tsx
```

Expected: FAIL missing UI.

- [ ] **Step 4: Implement panel** under Workspace detail — load integrations, filter `workspace_id === workspace.id`, ensure three kinds displayed (if API lag, show placeholders). Modal: Select mode first; then fields; PATCH `{ mode, enabled, config }`.

- [ ] **Step 5: Run tests + fix coverage thresholds if needed**

- [ ] **Step 6: Commit**

```bash
git add apps/web/src/pages/WorkspaceDetailPage.tsx apps/web/src/pages/WorkspaceDetailPage.test.tsx apps/web/src/components/integrations/ apps/web/src/locales/
git commit -m "feat: show workspace integration slots on workspace detail"
```

---

### Task 8: `/integrations` inventory alignment

**Files:**
- Modify: `apps/web/src/pages/IntegrationsPage.tsx`
- Modify: `apps/web/src/pages/IntegrationsPage.test.tsx`
- Locales as needed

**Interfaces:**
- Produces: filters Scope=Global|Workspace, Kind, Status; Remove “create workspace jira” as primary path — “Add” only creates missing **global** kinds; workspace rows opened via Configure (mode-aware).

- [ ] **Step 1: Failing tests** for filter + configure workspace slot with mode Inherit.

- [ ] **Step 2: Implement filters + mode-aware editor** (shared with Task 7 component if extracted).

- [ ] **Step 3: Run**

```bash
cd apps/web && npx vitest --run src/pages/IntegrationsPage.test.tsx
```

- [ ] **Step 4: Commit**

```bash
git add apps/web/src/pages/IntegrationsPage.tsx apps/web/src/pages/IntegrationsPage.test.tsx apps/web/src/locales/
git commit -m "feat: align integrations inventory with workspace slots"
```

---

### Task 9: Docs + gate

**Files:**
- Modify: `docs/03-data-model.md` (integrations + `mode`)
- Modify: `docs/04-api-spec.md` (mode, slot_status, resolve behaviour on test)
- Modify: `docs/integrations/README.md` (workspace slots + soft-skip)
- Modify: `docs/features/support-levels-and-workspaces.md` if it still says override-only
- Append status note on a new epic file **or** backlog story stubs (optional): `backlog/epics/EPIC-14-workspace-integration-slots.md` linking the spec/plan — only if the repo loop expects an epic; otherwise PR description is enough.

- [ ] **Step 1: Update docs to match behaviour** (no TBDs).

- [ ] **Step 2: Run full gate**

```bash
make lint type test
```

Expected: all green; coverage ≥ 90%.

- [ ] **Step 3: Commit**

```bash
git add docs/
git commit -m "docs: document workspace integration slots and resolve rules"
```

---

## Spec coverage check

| Spec requirement | Task |
|------------------|------|
| Always three slots per workspace | 1, 2, 6 |
| Mode inherit \| custom | 1, 6 |
| Resolver order + soft-skip | 3, 4 |
| Escalate/handoff same resolve | 5 |
| Workspace UI + `/integrations` | 7, 8 |
| Inheritance obvious UX | 7, 8 |
| Secrets rules (strip / full custom) | 6 |
| Docs | 9 |
| No hard-fail ingest | 4 |

## Placeholder / consistency review

- Reason codes: `slot_disabled`, `slot_missing`, `custom_incomplete`, `no_global`, `global_disabled` — used consistently in Tasks 3–5.
- `UpdateIntegration` gains `mode` in Task 2; Task 6 call sites must match.
- Slack Inherit overlays: **none** in this plan (explicit YAGNI per spec).
- Timeline kind: `integration_skipped`.
