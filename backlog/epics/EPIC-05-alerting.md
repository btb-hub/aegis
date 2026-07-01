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

- **Status:** Done
- **Depends on:** AEG-033
- **PRD:** REQ-ALERT-03
- **Acceptance:**
  - [x] `group_by=severity` and `group_by=label:key`
  - [x] Returns bucket counts

**Merged:** PR #10 (`feat/alerting-AEG-034-grouping-api`).

---

### AEG-035 — Saved views

- **Status:** In Review
- **Depends on:** AEG-033
- **PRD:** REQ-ALERT-04
- **Acceptance:**
  - [x] CRUD saved views with filter JSON
  - [x] Optional team share flag

**Agent plan (2026-06-26):** Migration `000008_saved_views`; CRUD `/api/v1/saved-views`; owner-only update/delete; `shared` visible to all users.

---

### AEG-036 — Inline analytics on alert slice

- **Status:** In Review
- **Depends on:** AEG-033
- **PRD:** REQ-ALERT-05
- **Acceptance:**
  - [x] Endpoint or embed in list response: severity counts, top labels

**Agent plan (2026-06-26):** `include_analytics=true` on list returns `by_severity`, `by_status`, `top_labels` for current filter slice.

---

### AEG-037 — CSV export

- **Status:** In Review
- **Depends on:** AEG-033
- **PRD:** REQ-ALERT-06
- **Acceptance:**
  - [x] Streamed CSV with current filters
  - [x] Does not load full result in memory

**Agent plan (2026-06-26):** `GET /alerts/export` reuses list filters; `StreamAlertsCSV` batches 500 rows per query.

---

### AEG-038 — Alert workspace UI — list and filters

- **Status:** In Review
- **Depends on:** AEG-033, AEG-034, AEG-054, AEG-055, AEG-056, AEG-058
- **PRD:** REQ-ALERT-01, REQ-ALERT-03
- **Acceptance:**
  - [x] Filter bar, paginated table, group-by toggle

---

### AEG-039 — Alert workspace UI — saved views and export

- **Status:** In Review
- **Depends on:** AEG-035, AEG-037, AEG-038
- **PRD:** REQ-ALERT-04, REQ-ALERT-06
- **Acceptance:**
  - [x] Save/load views
  - [x] Export button triggers download
