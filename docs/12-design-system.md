# Design system

Visual reference: [`design_system.html`](./design_system.html) (open in a browser). This file is the
machine-readable summary for agents. The HTML canvas records the original system. The
[October 2026 refinement review](./ui-refinement-review.html) documents the updated shell,
readability, and interaction decisions below, including the outstanding rendered-review gate.
Use the shared components and semantic tokens for current implementation.

## Scope

All web UI in `apps/web` follows this system. Chat page templates (Slack, eXpress) use the same
severity colors and tone where Block Kit / bubble layout allows.

| Surface | Uses design system? |
|---------|---------------------|
| Web app (`apps/web`) | Yes — tokens, components, patterns |
| Setup wizard | Yes |
| Analytics dashboards | Yes |
| Slack / eXpress pages | Severity colors + voice; layout is provider-specific |
| API error JSON | No — stable `code`; UI maps to localized copy |

## Voice and tone

Direct and calm under pressure. Status lines name the actor and time (`Acknowledged by Maya · 14:02`).
No decorative bounce or playful motion during incidents.

## Foundations

### Color

| Token | Role | Value (reference) |
|-------|------|-------------------|
| Neutrals | Text, surfaces, borders | Zinc ramp — text 900, secondary text 600, separators 200, field boundaries 500 |
| Canvas / surface | Working background / opaque content | `#F5F6F8` / `#FFFFFF` |
| Navigation | Shell / primary text / secondary text | `#172235` / `#F4F7FC` / `#B5C2D6` |
| Aegis Blue | Primary actions, links, focus, "now" indicator | `#2563EB` — use sparingly |
| P1 Critical | Highest severity | Red family (see canvas) |
| P2 High | | `#C2410C` on orange 50 |
| P3 Moderate | | `#92400E` on amber 50 |
| P4 Low | | Blue family (`#2563EB` on `#EFF6FF`) |
| Resolved | Success / cleared | `#15803D` on `#F0FDF4` |

### Typography

| Role | Font | Use |
|------|------|-----|
| UI / body | IBM Plex Sans | Interface copy, incident detail, descriptions |
| Data / labels | IBM Plex Mono | IDs, metrics, overlines, timestamps |
| Display | Plex Sans 32/40 · 600 | Display text |
| H1 | 24/32 · 600 | Page titles |
| H2 | 18/26 · 600 | Section headings |
| H3 | 14/20 · 600 | Subheads |
| Body | 14/21 · 400 | Default reading size |
| Fields | Plex Sans 16/24 · 400 | Input and select values, including phones |
| Record labels | Plex Sans 14/20 · 500 | Table headers, severity, status, navigation, actions |
| Caption | 12/16 · 400 | Helper text |
| Overline | 11/16 · 500 mono | Section labels, uppercase |

### Spacing and layout

- **Base unit:** 4px (`sm` 8, `md` 16, `lg` 32).
- **Desktop grid:** 12 columns, 24px gutter, 1280px max width.
- **Mobile grid:** 4 columns, 16px gutter and margin.
- **Shell:** 240px sticky, independently scrollable dark sidebar on desktop. Account and locale
  controls live at its foot. No separate desktop utility header. Below 1024px, a 64px brand/menu
  header expands navigation in document flow, with Escape returning focus to its toggle.
- **Incident workspace:** queue and detail side by side at 1280px and above. Below that, explicit
  selection opens detail with a Back to incidents action, preserving filter, row focus, and history.
- **Density:** table row 44px; compact 36px.

### Radius and elevation

- **Radius:** `sm` card, `md` popover, `lg` modal (see canvas for px values).
- **Elevation:** subtle overlays only — no heavy drop shadows on data tables.

### Motion

| Duration | Easing | Use |
|----------|--------|-----|
| 100ms | ease-out | Micro-feedback (hover, focus ring) |
| 160ms | cubic(.2,0,0,1) | Popovers, small panels |
| 220ms | ease-out | Modals, drawers |
| instant | — | Severity changes, ack state |

Never use decorative bounce. Motion supports orientation, not delight.

## Components

Implement once under `apps/web/src/components/ui/` (or equivalent) and reuse across features.

| Component | Notes |
|-----------|-------|
| **Buttons** | Primary, Secondary, Ghost; minimum height 36 desktop / 44 phone; radius 6; label 14/500; **one primary per current task** |
| **Inputs & fields** | Minimum height 40 desktop / 44 phone; Aegis Blue focus; visible labels and associated hint/error text; unique IDs |
| **Severity & label tags** | P1–P4 + neutral; mono ID optional |
| **Toasts** | Page receipts at top-right; when a dialog is open, feedback and recovery controls stay inside that dialog; return to page on close |
| **Incident card** | Severity stripe, title, team, relative time, primary action |
| **Banner & empty state** | Tell user what to do next; no illustration clutter |
| **Data table** | Sortable headers, row hover, compact mode; performant at 10k rows (NFR-2); structured phone rows preserve essential fields, with contained scrolling for two-dimensional comparisons |
| **Sidebar navigation** | Real route links; named landmark; active item; Operations / Organization / Administration groups; mobile disclosure; skip link |
| **Modal / dialog** | Native modal surface, inert background, initial focus, contained Tab sequence, Escape, visible close, and focus restoration |

Full specs, states (hover, disabled), and examples: [`design_system.html`](./design_system.html).

## Patterns

Reusable compositions documented in the canvas **03 / Patterns** section — incident list row,
on-call strip, escalation timeline, filter bar. Prefer these over one-off layouts.

## Pickers and localization

**04 / Pickers & Localization** in the canvas covers date/range pickers with `en` and `ru` weekday
labels. Pickers follow active locale and `Intl` formatting per [`11-localization.md`](./11-localization.md).

## Implementation

### Tailwind theme

Map design tokens to `apps/web/tailwind.config.ts` CSS variables (or `@theme` if using Tailwind v4).
Do not hard-code hex values in feature components — use semantic names (`severity-p1`, `accent`,
`surface-muted`, etc.).

### Icons

1.5px stroke, 24px grid. Lucide (or equivalent) at consistent weight.

### Storybook

Living component catalog in `apps/web` (Storybook 8+). Required before Phase 1 UI work.

| What | Where |
|------|-------|
| Run locally | `cd apps/web && npm run storybook` → `http://localhost:6006` |
| Stories | `apps/web/src/components/**/*.stories.tsx` |
| CI | `npm run build-storybook` must pass |

Every base component story covers: default, hover/focus (via pseudo or action), disabled where
applicable, and severity variants (P1–P4 for tags). Shell stories wrap content in `AppShell` with
`en` and `ru` locale decorators.

New shared components **must** ship with a Storybook story in the same PR (REQ-DS-04).

### Agent workflow

1. **Before UI work:** open [`design_system.html`](./design_system.html) for tokens; browse Storybook for component states.
2. **Reuse** shared components; extend the library if a pattern is missing — add a Storybook story with the component.
3. **PR checklist:** new UI matches tokens, typography, and component rules; one primary button per
   view; severity colors from the system; Storybook story for new shared components.
4. **Strings:** still ship `en` + `ru` per [`11-localization.md`](./11-localization.md).

## Responsive review

Phone triage uses a native status filter and a focused detail view with a return path. Alert rows retain severity, title, status, and received time; advanced filters use a disclosure, and analytics follow the records. On-call schedules use a daily phone agenda with UTC times while larger screens retain the timeline. Workspace integrations keep Configure actions visible in labeled phone rows. Controls, pagination, status labels, and headings must reflow with enlarged text. See the [rendered review](./ui-refinement-review.html) for captures, tested states, and remaining boundaries.

## References

- Visual canvas: [`design_system.html`](./design_system.html)
- Localization: [`11-localization.md`](./11-localization.md)
- Architecture (web stack): [`02-architecture.md`](./02-architecture.md)
- Story: [AEG-055](../backlog/epics/EPIC-01-foundation.md), [AEG-056](../backlog/epics/EPIC-01-foundation.md) (Storybook) in EPIC-01
