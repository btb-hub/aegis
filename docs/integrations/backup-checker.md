# Backup checker contract

**Status:** draft v1 (AEG-109). Aegis implements this contract; the backup-checker service must
match it or the differences must be agreed and this page updated first.

The backup-checker verifies each backup on a schedule: it compares the copies in every S3
location, restores the backup into a scratch database, and compares table row counts with the
backup manifest. Aegis reads the results for the **Backup status** page and stores on-call
reviews itself. Aegis never stores checker results; it reads them on each page load.

## Connection

Admins set these values in **Settings → Backups** (database settings, not env):

| Key | Meaning |
| --- | --- |
| `BACKUP_CHECKER_URL` | Base URL, `http` or `https`, no path, query or credentials, for example `http://backup-checker:8080` |
| `BACKUP_CHECKER_TOKEN` | Service token, at least 16 bytes, no spaces. Secret: never returned by the Settings API |
| `BACKUP_STALE_AFTER` | Go duration after which a check is marked stale. Empty means `26h` |

Aegis sends `Authorization: Bearer <token>` and `Accept: application/json`, waits at most 10
seconds, does not follow redirects, and reads at most 1 MiB per response. The checker should be
reachable only from the private network.

## Check object

```json
{
  "id": "chk_20261125_nish",
  "backup": "nish",
  "status": "failed",
  "trigger": "scheduled",
  "started_at": "2026-11-25T03:00:00Z",
  "finished_at": "2026-11-25T03:20:00Z",
  "replicas_match": true,
  "restore_ok": true,
  "manifest_counts_match": false,
  "message": "orders: 10412 rows, manifest 10420"
}
```

| Field | Rules |
| --- | --- |
| `id` | Required, 1–128 characters, unique per run, no whitespace, `/`, `?`, `#` or `%` |
| `backup` | Required, `^[a-z0-9][a-z0-9._-]{0,127}$` (for example `gamepass`, `nish`) |
| `status` | `running`, `passed` or `failed` |
| `trigger` | `scheduled`, `manual` or empty |
| `started_at` | Required RFC 3339 time |
| `finished_at` | Required when `status` is `passed` or `failed`; `null` while running |
| `replicas_match` | `true` when every S3 copy matches; `null` until checked |
| `restore_ok` | `true` when the test restore succeeded; `null` until checked |
| `manifest_counts_match` | `true` when restored row counts match the manifest; `null` until checked |
| `message` | Optional human-readable detail, up to 2000 characters |

Unknown extra fields are ignored, so the checker can add fields without breaking Aegis. A
response that breaks any rule above is rejected as a whole.

## Endpoints

| Method | Path | Success | Notes |
| --- | --- | --- | --- |
| GET | `/v1/checks` | `200 {"items": [Check]}` | Recent runs, at least the latest run of every backup, at most 1000 items. Aegis picks the latest per backup by `started_at` |
| GET | `/v1/checks/{id}` | `200 Check` | `404` when the run doesn't exist |
| POST | `/v1/checks/run` | `202 Check` with `status: "running"` | Body `{"backup": "<name>"}`. `404` for an unknown backup, `409` while a run for that backup is in progress. Used by AEG-112 |

## How Aegis reports checker errors

| Checker situation | Aegis API response |
| --- | --- |
| URL or token not saved | `503 BACKUP_CHECKER_NOT_CONFIGURED` |
| Network error, timeout, redirect or non-2xx other than 404/409 | `503 BACKUP_CHECKER_UNAVAILABLE` |
| Body isn't one JSON value, exceeds 1 MiB or breaks the rules above | `502 BACKUP_CHECKER_INVALID_RESPONSE` |
| `404` | `404 NOT_FOUND` |
| `409` | `409 CONFLICT` |

## Tests

Recorded fixtures live in [`pkg/backupchecker/testdata`](../../pkg/backupchecker/testdata). CI
never calls a live checker.
