# EPIC-03 — Integrations

**Phase:** 2–3  
**Exit:** Jira + Slack + eXpress with test connection and ack.

---

### AEG-016 — Connector interface and registry

- **Status:** Blocked
- **Depends on:** AEG-007
- **PRD:** REQ-INT-01, REQ-INT-06
- **Acceptance:**
  - [ ] `TicketProvider` and `ChatProvider` interfaces in Go
  - [ ] Registry loads from `integrations` table
  - [ ] Failure in one provider does not panic worker

---

### AEG-017 — Jira ticket provider

- **Status:** Blocked
- **Depends on:** AEG-016
- **PRD:** REQ-INT-02
- **Acceptance:**
  - [ ] Create issue on incident open
  - [ ] Store `jira_issue_key` on incident
  - [ ] Recorded fixture tests; no live Jira in CI

---

### AEG-018 — Slack chat provider

- **Status:** Blocked
- **Depends on:** AEG-016
- **PRD:** REQ-INT-03, REQ-I18N-04
- **Acceptance:**
  - [ ] Block Kit page to user DM (copy from `pkg/i18n` per recipient locale)
  - [ ] Interactive ack callback verified with signing secret
  - [ ] Recorded fixture tests

---

### AEG-019 — eXpress chat provider

- **Status:** Blocked
- **Depends on:** AEG-018
- **PRD:** REQ-INT-04, REQ-I18N-04
- **Acceptance:**
  - [ ] BotX HTTP client sends bubble with ack action (localized via `pkg/i18n`)
  - [ ] `/link` bootstrap documented and API stub for binding huid
  - [ ] HMAC webhook handler
  - [ ] Recorded fixture tests

---

### AEG-020 — Test connection for integrations

- **Status:** Blocked
- **Depends on:** AEG-017, AEG-019
- **PRD:** REQ-INT-05
- **Acceptance:**
  - [ ] `POST /integrations/{id}/test` for each provider
  - [ ] Clear error messages on failure
  - [ ] Admin UI button (minimal)
