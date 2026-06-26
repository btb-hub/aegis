# EPIC-04 — Incidents

**Phase:** 2  
**Exit:** Alert → incident → Jira → Slack page → ack → escalation.

---

### AEG-021 — Routing rules

- **Status:** Blocked
- **Depends on:** AEG-009
- **PRD:** REQ-INC-04
- **Acceptance:**
  - [ ] `routing_rules` CRUD
  - [ ] Match labels to team by priority

---

### AEG-022 — Alert dedup and fingerprint

- **Status:** Blocked
- **Depends on:** AEG-007
- **PRD:** REQ-INC-03
- **Acceptance:**
  - [ ] Fingerprint from configurable label set
  - [ ] Open incident reuse within window

---

### AEG-023 — process_alert worker job

- **Status:** Blocked
- **Depends on:** AEG-021, AEG-022, AEG-014
- **PRD:** REQ-INC-02, REQ-INC-05, REQ-INC-06
- **Acceptance:**
  - [ ] Job creates incident, assigns on-call
  - [ ] Links alert via `incident_alerts`
  - [ ] Idempotent on retry

---

### AEG-024 — Incident state machine

- **Status:** Blocked
- **Depends on:** AEG-023
- **PRD:** REQ-INC-05
- **Acceptance:**
  - [ ] States: open, acknowledged, resolved
  - [ ] Valid transitions enforced in service layer

---

### AEG-025 — Timeline events

- **Status:** Blocked
- **Depends on:** AEG-024
- **PRD:** REQ-INC-11
- **Acceptance:**
  - [ ] Append-only timeline on create, ack, resolve, page, handoff
  - [ ] API `GET /incidents/{id}/timeline`

---

### AEG-026 — Wire Jira on incident create

- **Status:** Blocked
- **Depends on:** AEG-023, AEG-017
- **PRD:** REQ-INC-07
- **Acceptance:**
  - [ ] Worker calls Jira provider after incident create
  - [ ] Timeline records ticket key

---

### AEG-027 — Wire Slack page on incident create

- **Status:** Blocked
- **Depends on:** AEG-023, AEG-018
- **PRD:** REQ-INC-08
- **Acceptance:**
  - [ ] Page assignee via Slack
  - [ ] `notifications` row per attempt

---

### AEG-028 — Acknowledge from UI and Slack

- **Status:** Blocked
- **Depends on:** AEG-024, AEG-018
- **PRD:** REQ-INC-09
- **Acceptance:**
  - [ ] `POST /incidents/{id}/acknowledge`
  - [ ] Slack button triggers same service method
  - [ ] Timeline + state update

---

### AEG-029 — Resolve incident

- **Status:** Blocked
- **Depends on:** AEG-024
- **PRD:** REQ-INC-05
- **Acceptance:**
  - [ ] `POST /incidents/{id}/resolve`
  - [ ] Sets `resolved_at`, timeline event

---

### AEG-030 — Escalation timer (one step)

- **Status:** Blocked
- **Depends on:** AEG-028
- **PRD:** REQ-INC-10
- **Acceptance:**
  - [ ] Schedule `escalate_incident` job on open
  - [ ] Cancel on ack
  - [ ] One re-page or notify step

---

### AEG-031 — Incident list and detail UI

- **Status:** Blocked
- **Depends on:** AEG-025, AEG-028, AEG-029, AEG-054, AEG-055, AEG-056
- **PRD:** REQ-INC-11
- **Acceptance:**
  - [ ] List with status filters
  - [ ] Detail page: timeline, alerts, Jira link, ack/resolve buttons
