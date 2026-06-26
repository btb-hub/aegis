# Design system

Visual reference: [`design_system.html`](./design_system.html) (open in a browser). This file is the
machine-readable summary for agents; the HTML canvas is the source of truth for tokens, spacing, and
component appearance.

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
| Neutrals | Text, surfaces, borders | Zinc ramp — text 900, muted 500, hairline 200, surface 50 |
| Aegis Blue | Primary actions, links, focus, "now" indicator | `#2563EB` — use sparingly |
| P1 Critical | Highest severity | Red family (see canvas) |
| P2 High | | Orange family |
| P3 Moderate | | Amber family |
| P4 Low | | Purple family |
| Resolved | Success / cleared | `#16A34A` on `#F0FDF4` |

### Typography

| Role | Font | Use |
|------|------|-----|
| UI / body | IBM Plex Sans | Interface copy, incident detail, descriptions |
| Data / labels | IBM Plex Mono | IDs, metrics, overlines, timestamps |
| Display H1 | Plex Sans 32/40 · 600 | Page titles |
| H2 | 24/32 · 600 | Section headings |
| H3 | 18/26 · 600 | Subheads |
| Body | 14/21 · 400 | Default reading size |
| Caption | 12/16 · 400 | Helper text |
| Overline | 11/16 · 500 mono | Section labels, uppercase |

### Spacing and layout

- **Base unit:** 4px (`sm` 8, `md` 16, `lg` 32).
- **Desktop grid:** 12 columns, 24px gutter, 1280px max width.
- **Mobile grid:** 4 columns, 16px gutter and margin.
- **Shell:** 240px fixed sidebar, 56px header.
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
| **Buttons** | Primary, Secondary, Ghost; height 36 (sm 30); radius 6; label 13/500; **one primary per view** |
| **Inputs & fields** | Clear focus ring (Aegis Blue); labels above; errors below field |
| **Severity & label tags** | P1–P4 + neutral; mono ID optional |
| **Toasts** | Match action verb from button; auto-dismiss; stack from top-right |
| **Incident card** | Severity stripe, title, team, relative time, primary action |
| **Banner & empty state** | Tell user what to do next; no illustration clutter |
| **Data table** | Sortable headers, row hover, compact mode; performant at 10k rows (NFR-2) |
| **Sidebar navigation** | Active item + count badge; collapses on mobile |
| **Modal / dialog** | Focus trap; explicit Primary / Secondary actions |

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

## References

- Visual canvas: [`design_system.html`](./design_system.html)
- Localization: [`11-localization.md`](./11-localization.md)
- Architecture (web stack): [`02-architecture.md`](./02-architecture.md)
- Story: [AEG-055](../backlog/epics/EPIC-01-foundation.md), [AEG-056](../backlog/epics/EPIC-01-foundation.md) (Storybook) in EPIC-01
