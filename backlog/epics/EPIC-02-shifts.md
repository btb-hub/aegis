# EPIC-02 — Shifts

**Phase:** 1  
**Exit:** Who's on call now is correct, including DST and overrides.

---

### AEG-009 — Teams and memberships API

- **Status:** Done
- **Depends on:** AEG-005, AEG-056
- **PRD:** REQ-SHIFT-01
- **Acceptance:**
  - [x] CRUD teams and team members
  - [x] Admin-only mutations
  - [x] Tests for authz

**Plan:** Migration for `teams` + `team_memberships`; store + service + `/api/v1/teams` handlers; session middleware; `RequireAdmin` on mutations; handler/service tests for authz.

---

### AEG-010 — Schedule and layer model

- **Status:** Done
- **Depends on:** AEG-009
- **PRD:** REQ-SHIFT-02
- **Acceptance:**
  - [x] Migration for schedules, schedule_layers
  - [x] API create/update weekly rotation with timezone
  - [x] Validation for participants list

**Plan:** Migration `schedules` + `schedule_layers`; store/service/handlers on `/api/v1/teams/{id}/schedules`; validate IANA timezone, weekly handoff, participants ⊆ team members; admin-only mutations.

---

### AEG-011 — Overrides API

- **Status:** Done
- **Depends on:** AEG-010
- **PRD:** REQ-SHIFT-03
- **Acceptance:**
  - [x] Create/delete overrides for team
  - [x] Override wins in resolution tests

**Plan:** Migration `overrides`; store/service/handlers on `/api/v1/teams/{id}/overrides`; validate team member + time range; enqueue `materialise_oncall`; override precedence covered in `pkg/oncall` tests.

---

### AEG-012 — On-call resolution engine

- **Status:** Done
- **Depends on:** AEG-011
- **PRD:** REQ-SHIFT-04
- **Acceptance:**
  - [x] Pure function: who is on call at instant T
  - [x] Unit tests for DST spring-forward and fall-back

**Plan:** `pkg/oncall` with `WhoAt` + `Materialise`; weekly rotation, layer priority, override precedence; DST tests for `America/New_York`.

---

### AEG-013 — On-call materialisation job

- **Status:** Done
- **Depends on:** AEG-012
- **PRD:** REQ-SHIFT-05
- **Acceptance:**
  - [x] Job `materialise_oncall` writes `on_call_slots`
  - [x] Triggered on schedule/override change + nightly
  - [x] 90-day horizon

**Plan:** Migration `on_call_slots`; `Store.MaterialiseOnCallForTeam`; worker `MaterialiseProcessor`; enqueue on schedule/override mutations; nightly goroutine in worker.

---

### AEG-014 — Current on-call API

- **Status:** Done
- **Depends on:** AEG-013
- **PRD:** REQ-SHIFT-07
- **Acceptance:**
  - [x] `GET /teams/{id}/on-call/current` returns user(s)
  - [x] Uses materialised slots

**Plan:** `OnCallService` + handler for `/on-call/current` and `/on-call/calendar` backed by `on_call_slots`.

---

### AEG-015 — Shifts calendar UI

- **Status:** Done
- **Depends on:** AEG-014, AEG-054, AEG-055, AEG-056
- **PRD:** REQ-SHIFT-06
- **Acceptance:**
  - [x] Month calendar with rotations and overrides
  - [x] "On call now" banner on team page
  - [x] Vitest tests for key components
  - [x] Storybook stories for OnCallBanner, ShiftsCalendar, TeamShiftsPage (en/ru)

**Plan:** `OnCallBanner`, `ShiftsCalendar`, `TeamShiftsPage` with i18n; Vitest coverage for banner + calendar.

**Post-MVP (EPIC-09):** Presentational components from AEG-015 are wired to live APIs and admin schedule/override UI in [AEG-068](../EPIC-09-teams-users-shifts.md)–[AEG-069](../EPIC-09-teams-users-shifts.md); demo fixtures removed from the `/shifts` route.
