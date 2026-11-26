# Backup status / Статус бэкапов

On-call engineers use **Backup status** (`/backups`, Operations navigation) to see whether every
backup (gamepass, nish and others) passed its latest automatic check and to record that a person
looked at it. Stories: [EPIC-16](../../backlog/epics/EPIC-16-backup-status.md).

## What the page shows

One row per backup, from the latest run in the backup-checker:

| Column | Source |
| --- | --- |
| Result | `passed`, `failed`, or `running` (shown as **Checking**), plus the checker's message |
| Checked | `finished_at`, or `started_at` while running |
| S3 copies | `replicas_match` |
| Restore | `restore_ok` |
| Counts vs manifest | `manifest_counts_match` |
| On-call review | Confirmed or Rejected, who, when and the reason; otherwise **Waiting for review** |
| Stale marker | Check older than `BACKUP_STALE_AFTER` (default 26 h) |

Check data is read from the checker on every load; Aegis only stores reviews. If the checker is
not connected, not responding, or sends data Aegis can't read, the page says so and what to do.
Admins get a link to **Settings → Backups**.

## Confirm and reject

- **Confirm** — the run passed and you checked it. Only `passed` runs can be confirmed.
- **Reject** — something is wrong. Needs a reason (up to 500 characters). Works on `passed` and
  `failed` runs.

A review belongs to one check run, never to the backup. When the next automatic run arrives the
row shows **Waiting for review** again. Each run has at most one review, and only the latest run
of a backup can be reviewed. If the page is out of date, Aegis refuses the review and reloads
the page.

Admins and members can review; viewers see the page read-only. Every review is written together
with an `audit_log` row (`backup.check_confirmed` or `backup.check_rejected`, actor, check id,
backup and reason).

## Recheck

Planned in AEG-112 (extra check through the checker) and AEG-113 (extra backup first, once the
backup trigger exists).

## References

- Checker contract: [`integrations/backup-checker.md`](../integrations/backup-checker.md)
- API: [`04-api-spec.md`](../04-api-spec.md#backup-status), [`openapi-backups.yaml`](../openapi-backups.yaml)
- Data model: [`03-data-model.md`](../03-data-model.md#backup-check-reviews-migration-25)
- How-to: [English](../how-to/backups.md), [Русский](../how-to/ru/backups.md)
