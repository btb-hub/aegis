# Backup status

The **Backup status** page shows the latest automatic check for every backup, such as gamepass and nish. On-call confirms each check after reviewing it, or rejects it with a reason.

## Connect the backup checker

1. Open **Settings → Backups** as an admin.
2. Enter the backup checker URL, for example `http://backup-checker.backup-checker.svc.cluster.local:8080`. Leave the token empty unless the checker requires one. Blank token fields keep the saved token.
3. Optionally set when a check counts as stale, for example `26h`. Empty means 26 hours.
4. Save, then open **Backup status** in the sidebar. If the page says the checker is not responding, check its address and network access.

## Review a check

Each row shows the result, when it was checked, whether the S3 copies match, whether the test restore worked, and whether row counts match the manifest. **Older than N h** means the checker has not reported for longer than the stale threshold.

- **Confirm** a passed check after you have looked at it.
- **Reject** a passed or failed check when something is wrong. Enter a short reason; it is shown on the page for the next on-call engineer.

A review applies to that check run only. When the next automatic run arrives, the row shows **Waiting for review** again. Admins and members can review; viewers can only look. Every review is recorded in the audit log.

See [Backup status](../features/backup-status.md) for the checker contract and API.
