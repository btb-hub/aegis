# EPIC-02 — Shifts

**Phase:** 1  
**Exit:** Who's on call now is correct, including DST and overrides.

---

### AEG-009 — Teams and memberships API

- **Status:** In Review
- **Depends on:** AEG-005, AEG-056
- **PRD:** REQ-SHIFT-01
- **Acceptance:**
  - [x] CRUD teams and team members
  - [x] Admin-only mutations
  - [x] Tests for authz

**Plan:** Migration for `teams` + `team_memberships`; store + service + `/api/v1/teams` handlers; session middleware; `RequireAdmin` on mutations; handler/service tests for authz.

---

### AEG-010 — Schedule and layer model

- **Status:** Ready
- **Depends on:** AEG-009
- **PRD:** REQ-SHIFT-02
- **Acceptance:**
  - [ ] Migration for schedules, schedule_layers
  - [ ] API create/update weekly rotation with timezone
  - [ ] Validation for participants list

---

### AEG-011 — Overrides API

- **Status:** Blocked
- **Depends on:** AEG-010
- **PRD:** REQ-SHIFT-03
- **Acceptance:**
  - [ ] Create/delete overrides for team
  - [ ] Override wins in resolution tests

---

### AEG-012 — On-call resolution engine

- **Status:** Blocked
- **Depends on:** AEG-011
- **PRD:** REQ-SHIFT-04
- **Acceptance:**
  - [ ] Pure function: who is on call at instant T
  - [ ] Unit tests for DST spring-forward and fall-back

---

### AEG-013 — On-call materialisation job

- **Status:** Blocked
- **Depends on:** AEG-012
- **PRD:** REQ-SHIFT-05
- **Acceptance:**
  - [ ] Job `materialise_oncall` writes `on_call_slots`
  - [ ] Triggered on schedule/override change + nightly
  - [ ] 90-day horizon

---

### AEG-014 — Current on-call API

- **Status:** Blocked
- **Depends on:** AEG-013
- **PRD:** REQ-SHIFT-07
- **Acceptance:**
  - [ ] `GET /teams/{id}/on-call/current` returns user(s)
  - [ ] Uses materialised slots

---

### AEG-015 — Shifts calendar UI

- **Status:** Blocked
- **Depends on:** AEG-014, AEG-054, AEG-055, AEG-056
- **PRD:** REQ-SHIFT-06
- **Acceptance:**
  - [ ] Month calendar with rotations and overrides
  - [ ] "On call now" banner on team page
  - [ ] Vitest tests for key components
