# EPIC-07 — Analytics & Setup

**Phase:** 6  
**Exit:** Five dashboard questions, setup wizard, a11y pass.

---

### AEG-045 — MTTA and MTTR API

- **Status:** Done
- **Depends on:** AEG-029
- **PRD:** REQ-AN-01, REQ-AN-03
- **Acceptance:**
  - [x] `/analytics/mtta` and `/analytics/mttr` time series
  - [x] `compare_previous` returns prior period

**Merged:** branch `feat/analytics-AEG-045-mtta-mttr-api` (Phase 6).

---

### AEG-046 — Noise and on-call load API

- **Status:** Done
- **Depends on:** AEG-027
- **PRD:** REQ-AN-01
- **Acceptance:**
  - [x] `/analytics/noise` top fingerprints
  - [x] `/analytics/on-call-load` pages per user

---

### AEG-047 — Handoff stats API

- **Status:** Done
- **Depends on:** AEG-044
- **PRD:** REQ-AN-01
- **Acceptance:**
  - [x] `/analytics/handoffs` count and median response time

**Merged:** PR #12 (`feat/l2-l3-phase-5`, AEG-044).

---

### AEG-048 — Overview dashboard API

- **Status:** Done
- **Depends on:** AEG-045, AEG-046, AEG-047
- **PRD:** REQ-AN-01
- **Acceptance:**
  - [x] `/analytics/overview` aggregates all five questions

---

### AEG-049 — Dashboard UI

- **Status:** Done
- **Depends on:** AEG-048, AEG-054, AEG-055, AEG-056
- **PRD:** REQ-AN-01, REQ-AN-02
- **Acceptance:**
  - [x] Recharts widgets for five questions
  - [x] Click-through to filtered lists

---

### AEG-050 — Setup wizard shell

- **Status:** Done
- **Depends on:** AEG-005, AEG-054, AEG-055, AEG-056
- **PRD:** REQ-AN-04
- **Acceptance:**
  - [x] Multi-step wizard route in web app
  - [x] Progress persisted in localStorage or server

---

### AEG-051 — Wizard integration steps

- **Status:** Done
- **Depends on:** AEG-050, AEG-020
- **PRD:** REQ-AN-04, REQ-INT-05
- **Acceptance:**
  - [x] Steps for Jira, Slack, eXpress with test connection
  - [x] OIDC smoke test step

---

### AEG-052 — Send test alert in wizard

- **Status:** Done
- **Depends on:** AEG-051, AEG-007
- **PRD:** REQ-AN-05
- **Acceptance:**
  - [x] Button posts sample payload to webhook
  - [x] Shows success when alert id returned

---

### AEG-053 — Accessibility pass and NFR-1 check

- **Status:** Done
- **Depends on:** AEG-049, AEG-052
- **PRD:** NFR-1
- **Acceptance:**
  - [x] Keyboard nav and focus order on main flows
  - [x] Focus rings and severity colors meet design system contrast ([`12-design-system.md`](../../docs/12-design-system.md))
  - [x] Language switcher reachable by keyboard; page `lang` reflects active locale
  - [x] Documented setup runbook meets ≤ 1 working day target
  - [x] axe or eslint-a11y clean on wizard + dashboard
