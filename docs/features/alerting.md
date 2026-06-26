# Feature: Alerting workspace

Implements PRD §3 (`REQ-ALERT-*`).

## Problem

Thousands of alerts; no fast way to filter, search, or export for postmortems.

## Solution

Indexed alert table with filters, full-text search, grouping, saved views, inline stats, CSV export.

## Filters

- Severity, status (`firing`/`resolved`), team, time range, arbitrary label key/value pairs.
- Filters compose as AND.

## Search

- `q` param searches `search_tsv` (title + body).
- Postgres `tsvector` + GIN on `labels` jsonb.

## Grouping

- `group_by=severity` or `group_by=label:team` returns bucket counts + representative rows.

## Saved views

- User saves current filter JSON under a name.
- Optional team-shared flag.

## Inline analytics

On current slice without extra round-trip:

- Count by severity
- Top 10 label values for selected key
- Firing vs resolved ratio

## Export

- `GET /alerts/export` streams CSV with same filters as list.
- Chunked response; no full memory load.

## Performance (NFR-2)

- Default indexes: `received_at DESC`, GIN `labels`, GIN `search_tsv`.
- Pagination required; max page size 100.

## References

- Epic: [`EPIC-05-alerting`](../../backlog/epics/EPIC-05-alerting.md)
