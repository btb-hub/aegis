# Data model

PostgreSQL 16. Migrations via golang-migrate. Queries via sqlc.

## Phase rollout

| Phase | Tables |
|-------|--------|
| 0 | `users`, `sessions`, `jobs`, `alerts` |
| 1 | `teams`, `team_memberships`, `schedules`, `schedule_layers`, `overrides`, `on_call_slots` |
| 2 | `routing_rules`, `incidents`, `incident_alerts`, `timeline_events`, `integrations`, `notifications` |
| 4 | `saved_views` (+ alert search indexes) |
| 5 | `handoffs` |
| 6 | `audit_log` (if not introduced earlier) |

## Core entities

### users

| Column | Type | Notes |
|--------|------|-------|
| id | uuid PK | |
| provider | text | `google`, `slack`, `express` |
| provider_sub | text | OIDC `sub`; unique with provider |
| email | text | |
| display_name | text | |
| role | text | `admin`, `member`, `viewer` |
| locale | text | `en` or `ru`; default `en` |
| slack_user_id | text nullable | For paging |
| express_user_huid | uuid nullable | For paging |
| created_at | timestamptz | |

No `password_hash`.

### sessions

| Column | Type | Notes |
|--------|------|-------|
| id | uuid PK | |
| user_id | uuid FK | |
| token_hash | text | HttpOnly cookie maps here |
| expires_at | timestamptz | |
| created_at | timestamptz | |

### jobs

| Column | Type | Notes |
|--------|------|-------|
| id | uuid PK | |
| kind | text | e.g. `process_alert`, `escalate_incident`, `materialise_oncall` |
| payload | jsonb | |
| status | text | `pending`, `running`, `done`, `failed` |
| run_at | timestamptz | For delayed/escalation jobs |
| attempts | int | |
| last_error | text nullable | |
| created_at | timestamptz | |
| updated_at | timestamptz | |

Index: `(status, run_at) WHERE status = 'pending'` for worker claims.

### alerts

| Column | Type | Notes |
|--------|------|-------|
| id | uuid PK | |
| fingerprint | text | Dedup key |
| status | text | `firing`, `resolved` |
| severity | text | |
| title | text | |
| body | text nullable | |
| labels | jsonb | GIN index |
| search_tsv | tsvector | Generated/stored for full-text |
| raw_payload | jsonb | Original webhook body |
| received_at | timestamptz | |

### teams, team_memberships

Standard org structure. Membership links `users` to `teams` with optional team role.

### schedules, schedule_layers, overrides

- `schedules`: team_id, name, timezone (IANA).
- `schedule_layers`: rotation config (participants, handoff time, type).
- `overrides`: user_id, start_at, end_at, replaces layer slot.

### on_call_slots

Materialised rows: `team_id`, `user_id`, `start_at`, `end_at`. Rebuilt by worker job on schedule change.

### incidents

| Column | Type | Notes |
|--------|------|-------|
| id | uuid PK | |
| team_id | uuid FK | |
| assignee_id | uuid FK nullable | Current on-call owner |
| status | text | `open`, `acknowledged`, `resolved` |
| severity | text | |
| title | text | |
| jira_issue_key | text nullable | |
| acknowledged_at | timestamptz nullable | |
| resolved_at | timestamptz nullable | |
| created_at | timestamptz | |

### timeline_events

Append-only: `incident_id`, `kind`, `actor_id`, `payload` jsonb, `created_at`.

### routing_rules

`team_id`, `match_labels` jsonb, `priority` int.

### integrations

`kind` (`jira`, `slack`, `express`), `config` jsonb (encrypted at rest optional MVP), `enabled` bool.

### notifications

Outbound page record: `incident_id`, `integration_id`, `status`, `external_ref`, `sent_at`.

### handoffs

`incident_id`, `from_user_id`, `to_user_id`, `from_team_id`, `to_team_id`, `reason`, `created_at`.

### saved_views

`owner_id`, `name`, `filter` jsonb, `shared` bool.

### audit_log

`actor_id`, `action`, `resource_type`, `resource_id`, `details` jsonb, `created_at`.

## Relationships (summary)

```text
users ──< team_memberships >── teams
teams ──< schedules ──< schedule_layers
teams ──< overrides
teams ──< on_call_slots >── users
alerts ──< incident_alerts >── incidents
incidents ──< timeline_events
incidents ──< notifications
incidents ──< handoffs
```

## References

- Architecture: [`02-architecture.md`](./02-architecture.md)
- API: [`04-api-spec.md`](./04-api-spec.md)
