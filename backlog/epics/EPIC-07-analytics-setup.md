# EPIC-07 — Analytics & Setup

**Phase:** 6  
**Exit:** Five dashboard questions, setup wizard, a11y pass.

---

### AEG-045 — MTTA and MTTR API

- **Status:** Blocked
- **Depends on:** AEG-029
- **PRD:** REQ-AN-01, REQ-AN-03
- **Acceptance:**
  - [ ] `/analytics/mtta` and `/analytics/mttr` time series
  - [ ] `compare_previous` returns prior period

---

### AEG-046 — Noise and on-call load API

- **Status:** Blocked
- **Depends on:** AEG-027
- **PRD:** REQ-AN-01
- **Acceptance:**
  - [ ] `/analytics/noise` top fingerprints
  - [ ] `/analytics/on-call-load` pages per user

---

### AEG-047 — Handoff stats API

- **Status:** Blocked
- **Depends on:** AEG-044
- **PRD:** REQ-AN-01
- **Acceptance:**
  - [ ] `/analytics/handoffs` count and median response time

---

### AEG-048 — Overview dashboard API

- **Status:** Blocked
- **Depends on:** AEG-045, AEG-046, AEG-047
- **PRD:** REQ-AN-01
- **Acceptance:**
  - [ ] `/analytics/overview` aggregates all five questions

---

### AEG-049 — Dashboard UI

- **Status:** Blocked
- **Depends on:** AEG-048, AEG-054, AEG-055, AEG-056
- **PRD:** REQ-AN-01, REQ-AN-02
- **Acceptance:**
  - [ ] Recharts widgets for five questions
  - [ ] Click-through to filtered lists

---

### AEG-050 — Setup wizard shell

- **Status:** Blocked
- **Depends on:** AEG-005, AEG-054, AEG-055, AEG-056
- **PRD:** REQ-AN-04
- **Acceptance:**
  - [ ] Multi-step wizard route in web app
  - [ ] Progress persisted in localStorage or server

---

### AEG-051 — Wizard integration steps

- **Status:** Blocked
- **Depends on:** AEG-050, AEG-020
- **PRD:** REQ-AN-04, REQ-INT-05
- **Acceptance:**
  - [ ] Steps for Jira, Slack, eXpress with test connection
  - [ ] OIDC smoke test step

---

### AEG-052 — Send test alert in wizard

- **Status:** Blocked
- **Depends on:** AEG-051, AEG-007
- **PRD:** REQ-AN-05
- **Acceptance:**
  - [ ] Button posts sample payload to webhook
  - [ ] Shows success when alert id returned

---

### AEG-053 — Accessibility pass and NFR-1 check

- **Status:** Blocked
- **Depends on:** AEG-049, AEG-052
- **PRD:** NFR-1
- **Acceptance:**
  - [ ] Keyboard nav and focus order on main flows
  - [ ] Focus rings and severity colors meet design system contrast ([`12-design-system.md`](../../docs/12-design-system.md))
  - [ ] Language switcher reachable by keyboard; page `lang` reflects active locale
  - [ ] Documented setup runbook meets ≤ 1 working day target
  - [ ] axe or eslint-a11y clean on wizard + dashboard
