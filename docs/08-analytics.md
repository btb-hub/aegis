# Analytics

Implements PRD §6 (`REQ-AN-*`). Answers the five north-star questions on one dashboard.

## Five questions

| # | Question | Widget |
|---|----------|--------|
| 1 | Are we getting faster? | MTTA + MTTR trends vs previous period |
| 2 | What's noisy? | Top alert fingerprints by volume |
| 3 | Is on-call load fair? | Pages / acks per user in range |
| 4 | How's L2→L3? | Handoff count, median time to L3 first response |
| 5 | How are we escalating? | % incidents escalated, time-to-escalation |

## Metrics definitions

### MTTA

Mean time from `incidents.created_at` to `acknowledged_at` for incidents acknowledged in range.

### MTTR

Mean time from `incidents.created_at` to `resolved_at` for incidents resolved in range.

### Noise

Group `alerts` by `fingerprint`, count in range, exclude resolved-only churn optional filter.

### On-call load

Count `notifications` sent per `user_id` where `kind=page`.

### Handoff

From `handoffs.created_at` to first timeline `acknowledged` by assignee after handoff.

## API

See [`04-api-spec.md`](./04-api-spec.md) — Analytics section.

Query params: `from`, `to` (ISO dates), `compare_previous=true` adds prior period of equal length.

## UI

- Overview dashboard with Recharts line/bar charts
- Click widget → filtered incident or alert list
- Export widget data CSV (Phase 6 story)

## References

- Epic: [`EPIC-07-analytics-setup`](../../backlog/epics/EPIC-07-analytics-setup.md)
