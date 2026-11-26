# EPIC-16 — Backup status

**Phase:** 15 (operations follow-up)
**Exit:** On-call engineers open **Backup status**, see the latest automatic check for every backup
(gamepass, nish, …), confirm or reject each check run, and run an extra check from the page. Every
decision is audited. A new automatic run waits for a fresh confirmation.

IDs continue from **AEG-109**. Pick one story at a time unless a human parallelizes.

## Problem

Backups are verified by the separate **backup-checker** service (replica comparison across S3
buckets, test restore, row counts against the manifest). Its results are not visible in Aegis, and
nobody records that a human looked at them. On-call has no place to say "checked, fine" or "not
fine, because …", and no way to ask for an extra check.

## Solution

1. **Connection** — Settings → Backups stores the backup-checker URL, service token and the
   staleness threshold (`BACKUP_STALE_AFTER`, default 26h).
2. **Status page** — `/backups` lists each backup with its latest check: result (passed, failed,
   running), when it ran, replicas match, restore succeeded, counts match the manifest, and stale
   marker.
3. **Review** — Confirm (passed run only) or Reject (finished run, comment required). Aegis stores
   the decision per check run, never per backup, so a new run needs a new review.
4. **Recheck** — run an extra check through `POST /v1/checks/run`; later, also trigger an
   extra backup first.

The checker contract (confirmed by the checker owner on Nov 26, 2026) lives in
[`docs/integrations/backup-checker.md`](../../docs/integrations/backup-checker.md).

**Out of scope:** editing backup schedules; alerting on failed checks (checker can send alerts
through the existing webhook); per-team visibility of backups.

**Proposed PRD additions:** REQ-BKP-01 (status page), REQ-BKP-02 (per-run review with audit),
REQ-BKP-03 (recheck). Role rule: admin and member may act; viewer is read-only.

---

### AEG-109 — Backup-checker connection and contract

- **Status:** In Review
- **Depends on:** none (database settings ship in migration 000022)
- **PRD:** REQ-BKP-01 (proposed)
- **Acceptance:**
  - [x] Settings has a **Backups** section with `BACKUP_CHECKER_URL`, `BACKUP_CHECKER_TOKEN`
        (secret, never returned) and `BACKUP_STALE_AFTER` (positive duration, default 26h)
  - [x] URL is validated like other connector URLs; token is optional (the in-cluster checker has
        no auth) and needs at least 16 bytes when set
  - [x] `pkg/backupchecker` client calls `GET /v1/checks`, `GET /v1/checks/{id}` and
        `POST /v1/checks/run {}` with an optional bearer token, 10s timeout, no redirects, 4 MiB
        limit, validated JSON; maps the checker's `target_id`, `status`, `sources_match`, `restore`
        and manifest inventory to the page columns
  - [x] Not configured → `503 BACKUP_CHECKER_NOT_CONFIGURED`; unreachable or non-2xx →
        `503 BACKUP_CHECKER_UNAVAILABLE`; malformed response → `502 BACKUP_CHECKER_INVALID_RESPONSE`
  - [x] Tests use `httptest` fixtures; no live checker

### AEG-110 — Backup status page

- **Status:** In Review
- **Depends on:** AEG-109
- **PRD:** REQ-BKP-01 (proposed)
- **Acceptance:**
  - [x] `GET /api/v1/backups` (any signed-in role) returns the latest check per backup, the review
        for that check run (if any) and `stale` when the check is older than `BACKUP_STALE_AFTER`
  - [x] `/backups` page in Operations navigation shows result, checked at, replicas, restore,
        manifest counts, review (who, when, comment) and a stale marker
  - [x] Checker errors show what happened and what to do (admins get a link to Settings)
  - [x] Strings in `en` and `ru`

### AEG-111 — Confirm and reject a check run

- **Status:** In Review
- **Depends on:** AEG-110
- **PRD:** REQ-BKP-02 (proposed)
- **Acceptance:**
  - [x] `POST /api/v1/backups/checks/{check_id}/confirm` and `…/reject` require admin or member and
        same origin; viewer gets 403
  - [x] Aegis re-reads the check from the checker; confirm needs `passed`, reject needs a finished
        run (`passed` or `failed`); otherwise 409
  - [x] Reject needs a comment (1–500 characters after trimming); empty → 400
  - [x] One review per check run (409 on a second); a new automatic run shows **Waiting for review**
  - [x] Review and `audit_log` row (`backup.check_confirmed` / `backup.check_rejected`, actor, check
        id, backup, comment) are written in one transaction
  - [x] Migration `000025_backup_check_reviews` (up and down)

### AEG-112 — Recheck (check only)

- **Status:** Ready
- **Depends on:** AEG-111
- **PRD:** REQ-BKP-03 (proposed)
- **Acceptance:**
  - [ ] **Recheck** (admin, member) calls `POST /v1/checks/run` and is audited
        (`backup.recheck_requested`). The checker's run endpoint takes `{}` and runs every target;
        a per-target run and the response body still need agreement with the checker owner
  - [ ] While the backup's latest check is `running`, Recheck is disabled in the UI and the API
        returns 409
  - [ ] The page polls `GET /v1/checks/{id}` until the run finishes and shows the result
  - [ ] The new run needs its own review

### AEG-113 — Recheck with an extra backup

- **Status:** Blocked (backup trigger API not available yet)
- **Depends on:** AEG-112, backup trigger in the backup tooling
- **PRD:** REQ-BKP-03 (proposed)
- **Acceptance:**
  - [ ] Recheck first starts an extra backup, then the check; the page shows both stages
  - [ ] Recheck stays disabled until both stages finish

**Plan (AEG-109–111, one PR):** `pkg/config/backups.go` + Settings section `backups`;
`pkg/backupchecker` client with `httptest` fixtures; migration `000025_backup_check_reviews` and
`pkg/db/backup_reviews.go` (review + audit in one tx); `service/backup.go` (latest run per backup,
stale flag, review rules) and `handler/backup.go` (session read, mutate + same-origin writes);
web `BackupsPage` with en/ru strings and Settings fields. Tests: Go unit tests per layer, DB test
against `DATABASE_URL`, Vitest for page and Settings. Branch: `codex/backup-status-page`.
