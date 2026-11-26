# Backup checker contract

**Status:** v1, taken from the backup-checker owner's API description (Nov 26, 2026). Aegis adapts
to the checker; when the checker changes, update this page and
[`pkg/backupchecker`](../../pkg/backupchecker) together.

The backup-checker verifies each backup target: it compares the copies in every S3 source,
restores the backup into a scratch database, and compares table row counts with the dump
manifest. Aegis reads the results for the **Backup status** page and stores on-call reviews
itself. Aegis never stores checker results; it reads them on each page load.

## Connection

In-cluster address: `http://backup-checker.backup-checker.svc.cluster.local:8080`.

Admins set these values in **Settings → Backups** (database settings, not env):

| Key | Meaning |
| --- | --- |
| `BACKUP_CHECKER_URL` | Base URL, `http` or `https`, no path, query or credentials |
| `BACKUP_CHECKER_TOKEN` | Optional. The checker has no authentication today; leave empty. When set (at least 16 bytes, no spaces) Aegis sends `Authorization: Bearer <token>`. Secret: never returned by the Settings API |
| `BACKUP_STALE_AFTER` | Go duration after which a check is marked stale. Empty means `26h` |

Aegis sends `Accept: application/json`, waits at most 10 seconds per call, does not follow
redirects, and reads at most 4 MiB per response. The checker must be reachable only from the
private network.

## Endpoints

| Method | Path | Used by Aegis | Notes |
| --- | --- | --- | --- |
| GET | `/healthz` | no | Liveness |
| GET | `/v1/checks` | Status page | JSON array of runs (summary fields only), at most 10,000 items |
| GET | `/v1/checks/{id}` | Status page, review | One run with sources, restore and manifest results. `404` when unknown |
| POST | `/v1/checks/run` | AEG-112 (not yet) | Body `{}`: starts a run of every target. Response body not agreed yet; Aegis ignores it. `409` is read as "already running" |
| GET | `/metrics` | no | Prometheus metrics |

## Run summary (`GET /v1/checks` items)

```json
{
  "id": "77c0712f-c349-4e7c-8a8d-cbeabafa876f",
  "target_id": "gamepass-prod-pg",
  "engine": "postgres",
  "status": "running",
  "started_at": "2026-11-26T15:51:48.082200Z",
  "finished_at": null,
  "sources_match": null,
  "error": null
}
```

| Field | Rules |
| --- | --- |
| `id` | Required, 1–128 characters, no whitespace, `/`, `?`, `#` or `%` (UUIDs today) |
| `target_id` | Required, `^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`. Aegis shows it as the backup name |
| `engine` | Optional, up to 64 characters, shown under the name |
| `status` | Required. See status mapping below |
| `started_at` | Required RFC 3339 time (fractional seconds allowed) |
| `finished_at` | Required once the run has finished; `null` while running |
| `sources_match` | `true` when every S3 source has the same object; `null` until compared |
| `error` | `null`, a string, or any JSON value (shown compactly) |

## Run details (`GET /v1/checks/{id}`)

The summary fields plus:

- `sources[]` — `name`, `endpoint`, `bucket`, `object_key`, `size_bytes`, `sha256`, `ok`, `error`.
  Aegis shows each source `error` as `<name>: <error>`.
- `restore` — `engine`, `ok`, `duration_seconds`, `error`, and `inventory`, a map from dump file
  to `{database, tables, manifest}`. `manifest` has `present`, `path`, `dump_sha256`,
  `dumped_at`, `mismatch_count`, `mismatches[]`, `extra_tables[]`, `missing_tables[]`.

See [`testdata/check_success.json`](../../pkg/backupchecker/testdata/check_success.json) for a
full example.

## How Aegis reads a run

| Page column | Derived from |
| --- | --- |
| Result | `status`, mapped below |
| S3 copies | `sources_match` |
| Restore | `restore.ok`; not checked when `restore` is missing |
| Counts vs manifest | **Mismatch** when any dump's manifest has `mismatch_count > 0` or non-empty `mismatches`, `extra_tables` or `missing_tables`. **Not checked** when any dump has no manifest (`present: false`) or there is no inventory. Otherwise **Match** |
| Message | Non-empty `error`, `restore.error` and source errors, joined with `; `, cut to 2000 characters |

Status mapping:

| Checker `status` | Aegis |
| --- | --- |
| `success`, `succeeded`, `passed`, `ok` | `passed` — can be confirmed |
| `running`, `pending`, `queued`, `started`, `in_progress` | `running` |
| `failed`, `failure`, `error`, `errored`, `timeout`, `cancelled`, `canceled` | `failed` |
| anything else | `failed` when `finished_at` is set, otherwise `running`; never `passed` |

The list doesn't include restore or manifest results, so on each load Aegis reads the details of
the latest run of every target (at most 4 calls at a time). Unknown extra fields are ignored.
A response that breaks any rule above is rejected as a whole.

## How Aegis reports checker errors

| Checker situation | Aegis API response |
| --- | --- |
| URL not saved | `503 BACKUP_CHECKER_NOT_CONFIGURED` |
| Network error, timeout, redirect or non-2xx other than 404/409 | `503 BACKUP_CHECKER_UNAVAILABLE` |
| Body isn't one JSON value, exceeds 4 MiB or breaks the rules above | `502 BACKUP_CHECKER_INVALID_RESPONSE` |
| `404` | `404 NOT_FOUND` |
| `409` | `409 CONFLICT` |

## Open questions for the checker owner

- `GET /v1/checks` has no limit or paging. If history grows past 10,000 runs or 4 MiB, the page
  fails; a `?limit=` or "latest per target" query would fix it.
- The full list of `status` values and the response of `POST /v1/checks/run` (needed for AEG-112).

## Tests

Recorded fixtures live in [`pkg/backupchecker/testdata`](../../pkg/backupchecker/testdata). CI
never calls a live checker.
