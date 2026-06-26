# EPIC-06 — L2↔L3

**Phase:** 5  
**Exit:** One-action handoff, shared timeline, bounce, analytics feed.

---

### AEG-040 — Handoff API and service

- **Status:** Blocked
- **Depends on:** AEG-025, AEG-014
- **PRD:** REQ-L2L3-01, REQ-L2L3-02
- **Acceptance:**
  - [ ] `POST /incidents/{id}/handoff` to L3 team
  - [ ] Reassigns to L3 on-call, records handoff row
  - [ ] Pages L3, optional Jira assignee update

---

### AEG-041 — Bounce to L2

- **Status:** Blocked
- **Depends on:** AEG-040
- **PRD:** REQ-L2L3-04
- **Acceptance:**
  - [ ] `POST /incidents/{id}/bounce` with required note
  - [ ] Reassigns to prior L2 owner

---

### AEG-042 — Shared timeline visibility

- **Status:** Blocked
- **Depends on:** AEG-040
- **PRD:** REQ-L2L3-03
- **Acceptance:**
  - [ ] L2 and L3 members see same timeline events
  - [ ] Test: no hidden events by role

---

### AEG-043 — Handoff UI

- **Status:** Blocked
- **Depends on:** AEG-040, AEG-041, AEG-054, AEG-055, AEG-056
- **PRD:** REQ-L2L3-01, REQ-L2L3-04
- **Acceptance:**
  - [ ] Hand off and bounce buttons on incident detail
  - [ ] Team picker for L3 target

---

### AEG-044 — Handoff analytics events

- **Status:** Blocked
- **Depends on:** AEG-040, AEG-028
- **PRD:** REQ-L2L3-05
- **Acceptance:**
  - [ ] Compute time to first L3 response from handoff
  - [ ] Exposed via analytics API for EPIC-07
