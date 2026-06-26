# EPIC-05 — Alerting

**Phase:** 4  
**Exit:** Fast filter, search, group, saved views, export.

---

### AEG-032 — Alert list indexes and search

- **Status:** Blocked
- **Depends on:** AEG-007
- **PRD:** REQ-ALERT-02, REQ-ALERT-07, NFR-2
- **Acceptance:**
  - [ ] GIN on `labels`, `search_tsv` on alerts
  - [ ] `GET /alerts?q=` full-text search
  - [ ] Benchmark or test asserting p95 target at 10k seed rows

---

### AEG-033 — Alert list filters and pagination

- **Status:** Blocked
- **Depends on:** AEG-032
- **PRD:** REQ-ALERT-01
- **Acceptance:**
  - [ ] Filter by severity, status, team, time range, labels
  - [ ] Pagination max 100 per page

---

### AEG-034 — Alert grouping API

- **Status:** Blocked
- **Depends on:** AEG-033
- **PRD:** REQ-ALERT-03
- **Acceptance:**
  - [ ] `group_by=severity` and `group_by=label:key`
  - [ ] Returns bucket counts

---

### AEG-035 — Saved views

- **Status:** Blocked
- **Depends on:** AEG-033
- **PRD:** REQ-ALERT-04
- **Acceptance:**
  - [ ] CRUD saved views with filter JSON
  - [ ] Optional team share flag

---

### AEG-036 — Inline analytics on alert slice

- **Status:** Blocked
- **Depends on:** AEG-033
- **PRD:** REQ-ALERT-05
- **Acceptance:**
  - [ ] Endpoint or embed in list response: severity counts, top labels

---

### AEG-037 — CSV export

- **Status:** Blocked
- **Depends on:** AEG-033
- **PRD:** REQ-ALERT-06
- **Acceptance:**
  - [ ] Streamed CSV with current filters
  - [ ] Does not load full result in memory

---

### AEG-038 — Alert workspace UI — list and filters

- **Status:** Blocked
- **Depends on:** AEG-033, AEG-034, AEG-054, AEG-055, AEG-056
- **PRD:** REQ-ALERT-01, REQ-ALERT-03
- **Acceptance:**
  - [ ] Filter bar, paginated table, group-by toggle

---

### AEG-039 — Alert workspace UI — saved views and export

- **Status:** Blocked
- **Depends on:** AEG-035, AEG-037, AEG-038
- **PRD:** REQ-ALERT-04, REQ-ALERT-06
- **Acceptance:**
  - [ ] Save/load views
  - [ ] Export button triggers download
