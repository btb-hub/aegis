# EPIC-06 — L2↔L3

**Phase:** 5  
**Exit:** One-action handoff, shared timeline, bounce, analytics feed.

**Plan (AEG-040–044):** Migration `handoffs`; `HandoffService` with handoff/bounce + `notify_handoff` worker job (page L3, Jira assignee); `GET /analytics/handoffs`; incident detail UI with team picker and bounce form. Timeline remains unfiltered by role.

---

**Merged:** PR #12 (`feat/l2-l3-phase-5`, AEG-040–044).

---

### AEG-040 — Handoff API and service

- **Status:** Done
- **Depends on:** AEG-025, AEG-014
- **PRD:** REQ-L2L3-01, REQ-L2L3-02
- **Acceptance:**
  - [x] `POST /incidents/{id}/handoff` to L3 team
  - [x] Reassigns to L3 on-call, records handoff row
  - [x] Pages L3, optional Jira assignee update

---

### AEG-041 — Bounce to L2

- **Status:** Done
- **Depends on:** AEG-040
- **PRD:** REQ-L2L3-04
- **Acceptance:**
  - [x] `POST /incidents/{id}/bounce` with required note
  - [x] Reassigns to prior L2 owner

---

### AEG-042 — Shared timeline visibility

- **Status:** Done
- **Depends on:** AEG-040
- **PRD:** REQ-L2L3-03
- **Acceptance:**
  - [x] L2 and L3 members see same timeline events
  - [x] Test: no hidden events by role

---

### AEG-043 — Handoff UI

- **Status:** Done
- **Depends on:** AEG-040, AEG-041, AEG-054, AEG-055, AEG-056
- **PRD:** REQ-L2L3-01, REQ-L2L3-04
- **Acceptance:**
  - [x] Hand off and bounce buttons on incident detail
  - [x] Team picker for L3 target

---

### AEG-044 — Handoff analytics events

- **Status:** Done
- **Depends on:** AEG-040, AEG-028
- **PRD:** REQ-L2L3-05
- **Acceptance:**
  - [x] Compute time to first L3 response from handoff
  - [x] Exposed via analytics API for EPIC-07
