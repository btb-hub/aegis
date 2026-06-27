# EPIC-04 — Incidents

**Phase:** 2  
**Exit:** Alert → incident → Jira → Slack page → ack → escalation.

---

### AEG-021 — Routing rules

- **Status:** Done
- **Depends on:** AEG-009
- **PRD:** REQ-INC-04
- **Acceptance:**
  - [x] `routing_rules` CRUD
  - [x] Match labels to team by priority

---

### AEG-022 — Alert dedup and fingerprint

- **Status:** Done
- **Depends on:** AEG-007
- **PRD:** REQ-INC-03
- **Acceptance:**
  - [x] Fingerprint from configurable label set (`ALERT_FINGERPRINT_LABELS`)
  - [x] Open incident reuse within window (`INCIDENT_DEDUP_WINDOW`)

---

### AEG-023 — process_alert worker job

- **Status:** Done
- **Depends on:** AEG-021, AEG-022, AEG-014
- **PRD:** REQ-INC-02, REQ-INC-05, REQ-INC-06
- **Acceptance:**
  - [x] Job creates incident, assigns on-call
  - [x] Links alert via `incident_alerts`
  - [x] Idempotent on retry

---

### AEG-024 — Incident state machine

- **Status:** Done
- **Depends on:** AEG-023
- **PRD:** REQ-INC-05
- **Acceptance:**
  - [x] States: open, acknowledged, resolved
  - [x] Valid transitions enforced in service layer

---

### AEG-025 — Timeline events

- **Status:** Done
- **Depends on:** AEG-024
- **PRD:** REQ-INC-11
- **Acceptance:**
  - [x] Append-only timeline on create, ack, resolve, page, handoff
  - [x] API `GET /incidents/{id}/timeline`

---

### AEG-026 — Wire Jira on incident create

- **Status:** Done
- **Depends on:** AEG-023, AEG-017
- **PRD:** REQ-INC-07
- **Acceptance:**
  - [x] Worker calls Jira provider after incident create
  - [x] Timeline records ticket key

---

### AEG-027 — Wire Slack page on incident create

- **Status:** Done
- **Depends on:** AEG-023, AEG-018
- **PRD:** REQ-INC-08
- **Acceptance:**
  - [x] Page assignee via Slack
  - [x] `notifications` row per attempt

---

### AEG-028 — Acknowledge from UI and Slack

- **Status:** Done
- **Depends on:** AEG-024, AEG-018
- **PRD:** REQ-INC-09
- **Acceptance:**
  - [x] `POST /incidents/{id}/acknowledge`
  - [x] Slack button triggers same service method
  - [x] Timeline + state update

---

### AEG-029 — Resolve incident

- **Status:** Done
- **Depends on:** AEG-024
- **PRD:** REQ-INC-05
- **Acceptance:**
  - [x] `POST /incidents/{id}/resolve`
  - [x] Sets `resolved_at`, timeline event

---

### AEG-030 — Escalation timer (one step)

- **Status:** Done
- **Depends on:** AEG-028
- **PRD:** REQ-INC-10
- **Acceptance:**
  - [x] Schedule `escalate_incident` job on open
  - [x] Cancel on ack
  - [x] One re-page or notify step

---

### AEG-031 — Incident list and detail UI

- **Status:** Done
- **Depends on:** AEG-025, AEG-028, AEG-029
- **PRD:** REQ-INC-11
- **Acceptance:**
  - [x] List with status filters
  - [x] Detail page: timeline, alerts, Jira link, ack/resolve buttons
