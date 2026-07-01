# EPIC-05 — Alerting

**Phase:** 4  
**Exit:** Fast filter, search, group, saved views, export.

---

### AEG-032 — Alert list indexes and search

- **Status:** Done
- **Depends on:** AEG-007
- **PRD:** REQ-ALERT-02, REQ-ALERT-07, NFR-2
- **Acceptance:**
  - [x] GIN on `labels`, `search_tsv` on alerts
  - [x] `GET /alerts?q=` full-text search
  - [x] Benchmark or test asserting p95 target at 10k seed rows

**Merged:** PR #7 (`feat/alerting-AEG-032-search-indexes`). Migration 000007 backfills `search_tsv` and adds `received_at DESC` index (GIN indexes from 000001).

---

### AEG-033 — Alert list filters and pagination

- **Status:** Done
- **Depends on:** AEG-032
- **PRD:** REQ-ALERT-01
- **Acceptance:**
  - [x] Filter by severity, status, team, time range, labels
  - [x] Pagination max 100 per page

**Merged:** PR #9 (`feat/alerting-AEG-033-filters-pagination`). Filters compose as AND; `CountAlerts` drives `total`; `team_id` resolves to `labels.team`.

---

### AEG-034 — Alert grouping API

- **Status:** In Review
- **Depends on:** AEG-033
- **PRD:** REQ-ALERT-03
- **Acceptance:**
  - [x] `group_by=severity` and `group_by=label:key`
  - [x] Returns bucket counts

**Agent plan (2026-06-26):** Parse `group_by` on `GET /api/v1/alerts`; when set, return `groups` with `key`, `count`, and `sample` alert per bucket. Reuse list filters. `GroupAlerts` in db with COUNT + DISTINCT ON sample query. Unit + integration tests; update API spec.

---

### AEG-035 — Saved views

- **Status:** Ready
- **Depends on:** AEG-033
- **PRD:** REQ-ALERT-04
- **Acceptance:**
  - [ ] CRUD saved views with filter JSON
  - [ ] Optional team share flag

---

### AEG-036 — Inline analytics on alert slice

- **Status:** Ready
- **Depends on:** AEG-033
- **PRD:** REQ-ALERT-05
- **Acceptance:**
  - [ ] Endpoint or embed in list response: severity counts, top labels

---

### AEG-037 — CSV export

- **Status:** Ready
- **Depends on:** AEG-033
- **PRD:** REQ-ALERT-06
- **Acceptance:**
  - [ ] Streamed CSV with current filters
  - [ ] Does not load full result in memory

---

### AEG-038 — Alert workspace UI — list and filters

- **Status:** Ready
- **Depends on:** AEG-033, AEG-034, AEG-054, AEG-055, AEG-056, AEG-058
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
