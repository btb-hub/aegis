# EPIC-10 — UI polish

**Phase:** 9  
**Exit:** Every web route matches [`docs/12-design-system.md`](../docs/12-design-system.md) and
[`docs/design_system.html`](../docs/design_system.html) — consistent headers, filter bars, tables,
and spacing. The Alerts workspace uses the composed **Search & Filter Bar** pattern instead of
misaligned ad-hoc cards.

## Problem

Feature epics shipped working UI quickly but skipped several design-system patterns. The Alerts page
is the worst offender (see audit below): saved-view controls, filter fields, and actions sit in
separate cards with mixed label alignment, uneven grid rows, and raw native controls. The same gaps
appear on other routes — inconsistent page titles, placeholder breadcrumbs, duplicated table markup,
and no shared `Select` / `Banner` / `PageHeader` components.

Phase 6 ([AEG-053](./EPIC-07-analytics-setup.md)) covered keyboard nav and contrast on wizard +
dashboard only. This epic is a **visual and compositional** pass — no new product behaviour.

## Design reference

- [`docs/12-design-system.md`](../docs/12-design-system.md) — tokens, typography, one-primary-per-view
- [`docs/design_system.html`](../docs/design_system.html) — **03 / Patterns → Search & Filter Bar**,
  **Data Table**, **Composed Screen · Desktop**
- Existing base components: `Button`, `Input`, `SeverityTag`, `StatusTag`, `Modal`, `Toast`,
  `PageBreadcrumb`

## Audit (issues to fix)

### Alerts (`AlertsPage`, `AlertFilterBar`, `AlertTable`) — priority

| Issue | Current | Target |
|-------|---------|--------|
| Filter layout | Two separate cards (saved views + filters); 4-column grid with label key/value stacked in one cell, breaking row height | Single **Search & Filter Bar** card: search full-width top row; filter fields in even grid; saved-view selector + name + share inline; active-filter chips + result count |
| Label alignment | Saved-view `<select>` uses inline label; filter selects use `font-medium` above; `Input` uses shared label-above — three styles on one page | All fields use shared `Input` / `Select` with label above |
| Native controls | Raw `<select>` and `<input type="checkbox">` with mixed `border-zinc-200` / `border-zinc-300` | Shared `Select` and `Checkbox` components |
| Actions placement | Export CSV in page header; Apply filters at bottom of filter card; Save view floating in saved-views card | Export secondary in filter bar toolbar (per canvas); Apply filters primary at bar end; one primary per view |
| Pagination | Raw `<button>` elements, detached from table card | Shared `Pagination` using `Button` ghost variant, aligned with table footer pattern |
| Breadcrumb | Hardcoded `shifts.demo_team` placeholder | Platform-consistent trail (`Platform / Alerts` or team-scoped when filter set) |
| Page title | `text-2xl` (24px) | H1 token: `text-[32px] leading-10 font-semibold` (or shared `PageHeader`) |
| Error state | Inline amber div | Shared `Banner` variant=warning |

### Incidents (`IncidentsPage`, `IncidentList`)

| Issue | Current | Target |
|-------|---------|--------|
| Page chrome | No breadcrumb; `text-3xl` title without `text-zinc-900` / subtitle spacing | `PageHeader` with breadcrumb + H1 + subtitle |
| Status filter | Custom pill buttons (`bg-zinc-900` active) | Segment control or ghost `Button` group per design system |
| Layout | `max-w-6xl` while shell content is full-width | `max-w-7xl` (1280px) content container, consistent with other pages |
| Empty state | Dashed border only | Shared empty-state pattern (message + next action) |

### Dashboard (`DashboardPage`)

| Issue | Current | Target |
|-------|---------|--------|
| Breadcrumb | Demo team placeholder | `Platform / Dashboard` |
| Compare toggle | Raw checkbox in header row | Styled checkbox via shared component; align with header actions slot |
| Metric cards | Section headings `text-sm` | H3 token (`text-lg font-semibold`) for card titles |
| Metric values | Plain text | Optional mono overline for labels per canvas analytics widgets |

### Teams & shifts (`TeamsPage`, `TeamDetailPage`, `TeamShiftsPage`, `ShiftsLandingPage`)

| Issue | Current | Target |
|-------|---------|--------|
| Title sizes | Mix of `text-2xl` and `text-3xl` across team routes | Unified `PageHeader` |
| Shifts landing | No breadcrumb; bare `text-3xl` heading | `PageHeader` + empty-state card |
| Team detail | Raw `<select>` for role; add-member panel uses `bg-zinc-50` ad-hoc | Shared `Select`; section card matches white surface pattern |
| Tables | Inline `<table>` duplicated | Shared `DataTable` where applicable |

### Integrations & setup (`IntegrationsPage`, `SetupWizardPage`)

| Issue | Current | Target |
|-------|---------|--------|
| Breadcrumb | Demo team placeholder on integrations and setup | Platform-scoped breadcrumb |
| Setup wizard | Multiple primary Save buttons visible in integrations step | One primary per visible sub-step; secondary for Test connection |
| Wizard progress | Pill steps OK but spacing tight vs canvas stepper | Match canvas step indicator spacing |

### Account (`AccountPage`)

| Issue | Current | Target |
|-------|---------|--------|
| Title | `text-3xl` without subtitle | `PageHeader` with subtitle |
| Language picker | Two primary buttons when locale selected (violates one-primary-per-view) | Toggle/segment pattern: one filled, one ghost |
| Layout | `max-w-2xl` centered — OK for form page | Keep narrow width; align section headings to H2/H3 tokens |

### App shell (`AppShell`)

| Issue | Current | Target |
|-------|---------|--------|
| Content width | `<main className="p-6">` with no max width | Optional `PageContent` wrapper: `mx-auto max-w-7xl w-full` (1280px grid) |

### Login (`LoginPage`)

| Issue | Current | Target |
|-------|---------|--------|
| Overall | Generally matches centered auth card pattern | Minor: ensure shadow/radius match canvas login spec during pass |

---

### AEG-072 — Shared form and feedback components

- **Status:** In Review
- **Depends on:** AEG-056
- **PRD:** REQ-DS-04; [`12-design-system.md`](../docs/12-design-system.md) Components table
- **Acceptance:**
  - [ ] `Select` — label above, height 36, radius 6, focus ring matching `Input`, disabled state
  - [ ] `Checkbox` — label beside, focus ring, used for share/compare toggles
  - [ ] `Banner` — variants: info, warning, error; role=alert/ status; no emoji
  - [ ] Storybook stories for each (default, disabled, variants)
  - [ ] Unit tests for render + keyboard focus
  - [ ] Replace at least one existing raw `<select>` usage as proof (can be in AEG-074 scope)

**Plan:** Add under `apps/web/src/components/ui/`. Map borders to `border-zinc-300`, focus to
`accent` ring — same as `Input`.

---

### AEG-073 — PageHeader and PageContent layout primitives

- **Status:** In Review
- **Depends on:** AEG-056
- **PRD:** [`12-design-system.md`](../docs/12-design-system.md) Typography + Spacing
- **Acceptance:**
  - [ ] `PageHeader` — breadcrumb slot, H1 (32/40 semibold), optional subtitle (14/21 muted),
        optional actions slot (right-aligned, flex wrap)
  - [ ] `PageContent` — `mx-auto w-full max-w-7xl` wrapper for main page body
  - [ ] Storybook: default, with actions, long title wrap, `en` + `ru`
  - [ ] Unit tests for slots and aria landmarks

**Plan:** Compose existing `PageBreadcrumb`. Export from `components/ui/`.

---

### AEG-074 — DataTable and Pagination components

- **Status:** In Review
- **Depends on:** AEG-056, AEG-072
- **PRD:** REQ-DS-04; design system Data Table pattern
- **Acceptance:**
  - [ ] `DataTable` — sortable header styling, 44px row height, hover row, compact prop
  - [ ] `Pagination` — prev/next ghost buttons, page indicator, total count slot; uses `Button`
  - [ ] Storybook + unit tests
  - [ ] Empty state row built-in (centered message, configurable)

**Plan:** Extract from `AlertTable` and `TeamsPage` table markup.

---

### AEG-075 — Alerts workspace layout polish

- **Status:** In Review
- **Depends on:** AEG-072, AEG-073, AEG-074
- **PRD:** REQ-ALERT-01, REQ-ALERT-04; design system Search & Filter Bar pattern
- **Acceptance:**
  - [ ] Single filter bar card replaces separate saved-view + filter cards
  - [ ] All fields use `Input` / `Select` / `Checkbox` — aligned grid, equal row heights
  - [ ] Saved-view load/save controls integrated in toolbar row
  - [ ] Export CSV secondary in filter bar or page header actions (not both)
  - [ ] Active filter chips + result count when filters applied (can be read-only chips)
  - [ ] `AlertTable` uses `DataTable` + `Pagination`
  - [ ] Error/empty states use `Banner` and table empty pattern
  - [ ] `PageHeader` + fix breadcrumb (no demo team placeholder)
  - [ ] Visual match to `design_system.html` filter bar (spacing, one primary Apply button)
  - [ ] Update `AlertsPage.test.tsx` and component tests; no regression in filter/save/export behaviour
  - [ ] Storybook story for composed Alerts filter bar (optional sub-story under alerts/)

**Plan:** Refactor `AlertFilterBar.tsx`, saved-view block in `AlertsPage.tsx`, wire shared components.
Reference user screenshot / composed screen in canvas.

---

### AEG-076 — Cross-page header and shell consistency

- **Status:** In Review
- **Depends on:** AEG-073
- **PRD:** [`12-design-system.md`](../docs/12-design-system.md)
- **Acceptance:**
  - [ ] All routes use `PageHeader` (except Login): Dashboard, Integrations, Setup, Teams, Team detail,
        Shifts landing, Team shifts, Account, Incidents
  - [ ] Remove `shifts.demo_team` placeholder breadcrumbs; use `Platform` or contextual parent
  - [ ] Unify H1 sizing to design-system H1 across routes
  - [ ] Wrap page bodies in `PageContent` via `AppShell` or per-page
  - [ ] Locale strings updated if breadcrumb labels change (`en` + `ru`)
  - [ ] Existing page tests updated

**Plan:** Mechanical refactor; one commit per page group if PR grows large.

---

### AEG-077 — Secondary pages visual polish

- **Status:** In Review
- **Depends on:** AEG-072, AEG-074, AEG-076
- **PRD:** [`12-design-system.md`](../docs/12-design-system.md)
- **Acceptance:**
  - [ ] **Incidents:** segment status filter, `PageHeader`, empty states
  - [ ] **Dashboard:** metric card headings H3, compare checkbox via `Checkbox`, breadcrumb fix
  - [ ] **Teams / Team detail / Integrations:** `DataTable`, `Select` for role picker
  - [ ] **Setup wizard:** one primary per integration sub-panel; step indicator spacing
  - [ ] **Account:** language toggle segment (no dual primary)
  - [ ] **Shifts landing:** `PageHeader` + list empty state
  - [ ] Tests green; Storybook updated for changed shared usage

**Plan:** After AEG-075 proves the pattern on Alerts, apply same primitives elsewhere.

---

## Dependency graph

```
AEG-056 (Storybook base — Done)
  ├── AEG-072 (Select, Banner, Checkbox)
  ├── AEG-073 (PageHeader, PageContent)
  └── AEG-074 (DataTable, Pagination) ── depends AEG-072
AEG-072 + AEG-073 + AEG-074 → AEG-075 (Alerts polish)
AEG-073 → AEG-076 (headers + shell)
AEG-072 + AEG-074 + AEG-076 → AEG-077 (other pages)
```

## Suggested pick order

1. AEG-072 ∥ AEG-073 (parallel)
2. AEG-074
3. AEG-075 (user-visible win on Alerts)
4. AEG-076
5. AEG-077

## Out of scope

- New alert/incident/team features or API changes
- Dark mode
- Mobile-specific layouts beyond existing responsive grids
- Rewriting incident detail or shifts calendar internals (only page chrome + controls)
- Chat connector template styling
