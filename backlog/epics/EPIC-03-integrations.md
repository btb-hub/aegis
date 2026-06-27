# EPIC-03 — Integrations

**Phase:** 2–3  
**Exit:** Jira + Slack + eXpress with test connection and ack.

---

### AEG-016 — Connector interface and registry

- **Status:** Done
- **Depends on:** AEG-007
- **PRD:** REQ-INT-01, REQ-INT-06
- **Acceptance:**
  - [x] `TicketProvider` and `ChatProvider` interfaces in Go
  - [x] Registry loads from `integrations` table
  - [x] Failure in one provider does not panic worker

**Plan (implemented):** `pkg/integrations` registry; worker loads enabled integrations; `ForEachTicket`/`ForEachChat` swallow per-provider errors.

---

### AEG-017 — Jira ticket provider

- **Status:** Done
- **Depends on:** AEG-016
- **PRD:** REQ-INT-02
- **Acceptance:**
  - [x] Create issue on incident open
  - [x] Store `jira_issue_key` on incident
  - [x] Recorded fixture tests; no live Jira in CI

---

### AEG-018 — Slack chat provider

- **Status:** Done
- **Depends on:** AEG-016
- **PRD:** REQ-INT-03, REQ-I18N-04
- **Acceptance:**
  - [x] Block Kit page to user DM (copy from `pkg/i18n` per recipient locale)
  - [x] Interactive ack callback verified with signing secret
  - [x] Recorded fixture tests

---

### AEG-019 — eXpress chat provider

- **Status:** Blocked
- **Depends on:** AEG-018
- **PRD:** REQ-INT-04
- **Acceptance:**
  - [ ] BotX HTTP client sends bubble with ack action
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