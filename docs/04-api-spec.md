# API specification

Base URL: `/api/v1`. JSON request/response unless noted. Errors: `{code, message, details}`.

OpenAPI schema generated from code in `apps/api` (future story).

## Auth

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/auth/{provider}/login` | — | Redirect to OIDC (`google`, `slack`, `express`) |
| GET | `/auth/{provider}/callback` | — | OIDC callback; sets session cookie; redirects to `PUBLIC_URL` (`302`). Pass `?format=json` for JSON user body instead. |
| POST | `/auth/logout` | session | Invalidate session |
| GET | `/auth/me` | session | Current user profile (see below) |
| PATCH | `/auth/me` | session | Update profile fields (`locale`: `en` \| `ru`, `display_name`: non-empty string) |

`GET /auth/me` response includes `id`, `email`, `display_name`, `role`, `locale`, `provider` (primary sign-in provider), optional `avatar_url`, `slack_user_id`, `express_user_huid`, and `identities[]` (`provider`, `provider_sub`, `linked_at`). OIDC login links providers by email when the identity is new; profile fields are backfilled only when empty.

### Development only

Available when `DEV_AUTH_ENABLED=true` and `PUBLIC_URL` points at localhost. Disabled by default; must not be enabled in production.

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/auth/dev/status` | — | `{ "enabled": true \| false }` |
| GET | `/auth/dev/login` | — | Create dev user session; `302` to `PUBLIC_URL`. Query: `role` (`admin` \| `member` \| `viewer`, default from env), optional `redirect` (relative path). Returns `404` when disabled. |

## Health

| Method | Path | Description |
|--------|------|-------------|
| GET | `/healthz` | Liveness |
| GET | `/readyz` | Postgres connectivity |
| GET | `/metrics` | Prometheus metrics |

## Alerts (Phase 0–2)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/alerts/webhook` | webhook secret header | Ingest alert; `202` + `{id}` |
| GET | `/alerts` | session | List alerts; supports `q` full-text search on title/body (Phase 4, AEG-032) |
| GET | `/alerts/{id}` | session | Detail |
| GET | `/alerts/export` | session | CSV stream |

Query params for list: `severity`, `status`, `team_id`, `from`, `to`, `q` (search), `group_by`, `page`, `page_size`.

**Implemented (AEG-032):** `GET /alerts?q=` runs full-text search against `search_tsv` (title + body). Results are ordered by `received_at` descending.

**Implemented (AEG-033):** Filters compose as AND: `severity`, `status`, `team_id` (resolved to `labels.team` via team name), `from`/`to` (RFC3339 on `received_at`), repeatable `label=key:value` for arbitrary labels. Pagination via `page` (default 1) and `page_size` (default 100, max 100). Response includes `total`, `page`, and `page_size`.

**Implemented (AEG-034):** `group_by=severity` or `group_by=label:<key>` returns grouped buckets instead of a flat list. Same filters apply. Response includes `group_by`, `groups` (each with `key`, `count`, and `sample` alert), and `total`.

**Implemented (AEG-036):** `include_analytics=true` adds an `analytics` object to the list response with `by_severity`, `by_status`, and `top_labels` (driven by optional `analytics_label_key`).

**Implemented (AEG-037):** `GET /alerts/export` streams CSV with the same filter query params as list (no pagination). Response is `text/csv` with `Content-Disposition: attachment`.

**Response (list):**

```json
{
  "items": [
    {
      "id": "uuid",
      "fingerprint": "sha256…",
      "status": "firing",
      "severity": "critical",
      "title": "HighCPU",
      "body": "CPU high on host-1",
      "labels": {"alertname": "HighCPU", "team": "platform"},
      "received_at": "2026-06-26T12:00:00Z"
    }
  ],
  "total": 42,
  "page": 1,
  "page_size": 100
}
```

**Response (grouped):**

```json
{
  "group_by": "severity",
  "groups": [
    {
      "key": "critical",
      "count": 5,
      "sample": {
        "id": "uuid",
        "fingerprint": "sha256…",
        "status": "firing",
        "severity": "critical",
        "title": "HighCPU",
        "body": "CPU high on host-1",
        "labels": {"alertname": "HighCPU", "team": "platform"},
        "received_at": "2026-06-26T12:00:00Z"
      }
    }
  ],
  "total": 10
}
```

## Teams & shifts (Phase 1)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/teams` | session | List teams |
| POST | `/teams` | session + admin | Create team |
| GET | `/teams/{id}` | session | Team detail |
| PATCH | `/teams/{id}` | session + admin | Update team (optional `workspace_id` to move team) |
| DELETE | `/teams/{id}` | session + admin | Delete team |
| GET | `/teams/{id}/members` | session | List memberships |
| POST | `/teams/{id}/members` | session + admin | Add member |
| PATCH | `/teams/{id}/members/{userId}` | session + admin | Update member `team_role` |
| DELETE | `/teams/{id}/members/{userId}` | session + admin | Remove member |

Create team body: `{"name": "Platform", "description": "optional"}`.

Add member body: `{"user_id": "uuid", "team_role": "member" | "lead"}` (defaults to `member`).

Member response includes `user_id`, `team_role`, `email`, `display_name`.

Update team body: `{"name": "Platform", "description": "optional", "support_tier": "l2" | "l3", "workspace_id": "uuid"}`.
Moving a team to another workspace is blocked with `409` when escalation paths would cross workspaces without `cross_workspace: true`. Response `details.blocked_teams` lists conflicting paths per team.

## Workspaces (Phase 11)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/workspaces` | session | List workspaces with `team_count` and `routing_rule_count` |
| GET | `/workspaces/{id}` | session | Workspace detail |
| POST | `/workspaces` | session + admin | Create workspace |
| PATCH | `/workspaces/{id}` | session + admin | Update workspace |
| DELETE | `/workspaces/{id}` | session + admin | Delete empty workspace |
| POST | `/workspaces/{id}/teams` | session + admin | Move teams to workspace (atomic bulk) |

Create/update body: `{"name": "Platform", "slug": "platform", "description": "optional"}`.

Assign teams body: `{"team_ids": ["uuid", "uuid"]}`. Returns `{"items": [team, ...]}`. On conflict, `409` with `details.blocked_teams`.

Delete rejects the default workspace (`00000000-0000-0000-0000-000000000001`) with `403`. Non-empty workspaces return `409` with usage counts in `details`.

## Users (Phase 8)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/users` | session + admin | Paginated user directory for member pickers and the Users admin page |
| PATCH | `/users/{id}` | session + admin | Body `{"role": "admin"\|"member"\|"viewer"}`. Conflicts: `last_admin`, `admin_emails_pinned`. |

Query params: `q` (search email and display name), `page` (default 1), `page_size` (default 100, max 100).

**Implemented (AEG-065):** Response items match `GET /auth/me` user fields: `id`, `email`, `display_name`, `role`, `avatar_url`, `slack_user_id`, `express_user_huid`, and `identities[]`. Sorted by `display_name` ascending. List items also include `role_pinned` (boolean), which reports whether the user's email is pinned to `admin` by the `ADMIN_EMAILS` config and therefore cannot be demoted from the UI.

```json
{
  "items": [
    {
      "id": "uuid",
      "email": "alice@example.com",
      "display_name": "Alice",
      "role": "member",
      "locale": "en",
      "provider": "google",
      "role_pinned": false,
      "identities": [
        {"provider": "google", "provider_sub": "…", "linked_at": "2026-07-03T12:00:00Z"}
      ]
    }
  ],
  "total": 1,
  "page": 1,
  "page_size": 100
}
```

**PATCH `/users/{id}`:** Changes a user's role. Returns the updated user in the same shape as a list item (including `role_pinned`). Rejects with `409 CONFLICT` and code `last_admin` when demoting the last remaining admin, and with `409` and code `admin_emails_pinned` when demoting a user whose email is pinned to admin by `ADMIN_EMAILS`.

Create/update schedule body:

```json
{
  "name": "Primary",
  "timezone": "Europe/Moscow",
  "rotation": {
    "handoff_weekday": 1,
    "handoff_time": "09:00",
    "participants": ["<user-uuid>"]
  }
}
```

`handoff_weekday`: 0 (Sunday) through 6 (Saturday). `participants` must be non-empty, unique, and team members.

| GET/POST | `/teams/{id}/schedules` | List/create schedules (create: admin) |
| GET/POST | `/teams/{id}/overrides` | List/create overrides (create: admin) |
| DELETE | `/teams/{id}/overrides/{oid}` | Delete override (admin) |
| GET | `/teams/{id}/on-call/current` | Current on-call user(s) |
| GET | `/teams/{id}/on-call/calendar` | Materialised slots in range (`from`, `to` RFC3339) |

Create override body: `{"user_id": "uuid", "start_at": "RFC3339", "end_at": "RFC3339"}`. `user_id` must be a team member; `end_at` must be after `start_at`.

Schedule and override changes materialise on-call slots synchronously for the team. The worker also runs a nightly `materialise_oncall` job for all teams with schedules.

| GET/PATCH/DELETE | `/teams/{id}/schedules/{sid}` | Schedule CRUD (mutations: admin) |

## Incidents (Phase 2)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/routing-rules` | List routing rules |
| POST | `/routing-rules` | Create routing rule (admin) |
| PATCH | `/routing-rules/{id}` | Update routing rule (admin) |
| DELETE | `/routing-rules/{id}` | Delete routing rule (admin) |
| GET | `/incidents` | List with filters |
| GET | `/incidents/{id}` | Detail + timeline |
| POST | `/incidents/{id}/acknowledge` | Ack from UI |
| POST | `/incidents/{id}/resolve` | Resolve |
| GET | `/incidents/{id}/timeline` | Timeline events |

## Integrations (Phase 2–3)

| Method | Path | Description |
|--------|------|-------------|
| GET/POST | `/integrations` | List/create |
| GET/PATCH/DELETE | `/integrations/{id}` | CRUD |
| POST | `/integrations/{id}/test` | Test connection |

## eXpress link (Phase 3)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/users/me/express-link-code` | Generate `/link` code (session) |
| POST | `/users/me/express-link` | Direct bind `express_user_huid` stub (session) |

## Callbacks (Phase 2–3)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/callbacks/slack/interactive` | Slack signature | Ack button |
| POST | `/callbacks/express/bot` | HMAC | eXpress bot events |

## Handoffs (Phase 5)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/incidents/{id}/handoff` | Hand to L3 team `{to_team_id, note}` |
| POST | `/incidents/{id}/bounce` | Bounce to L2 `{note}` (note required) |

**Implemented (AEG-040–041):** Handoff reassigns the incident to the target team's current on-call, records a `handoffs` row, appends a timeline event, and enqueues `notify_handoff` (pages L3 via chat connectors; updates Jira assignee when configured). Bounce reassigns to the prior L2 owner from the latest non-bounced handoff.

Handoff body:

```json
{
  "to_team_id": "uuid",
  "note": "optional context for L3"
}
```

## Analytics (Phase 6)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/analytics/overview` | Dashboard aggregates |
| GET | `/analytics/mtta` | MTTA series |
| GET | `/analytics/mttr` | MTTR series |
| GET | `/analytics/noise` | Top noisy fingerprints |
| GET | `/analytics/on-call-load` | Pages per user |
| GET | `/analytics/handoffs` | L2→L3 stats |

Query: `from`, `to`, `compare_previous` (bool).

**Implemented (AEG-044):** `GET /analytics/handoffs?from=&to=` (RFC3339) returns `count` and `median_response_seconds` (handoff to first L3 acknowledge).

**Implemented (AEG-045):** `GET /analytics/mtta` and `GET /analytics/mttr` return daily `series` buckets plus period `mean_seconds` and `count`. `compare_previous=true` adds a `previous` object for the prior range of equal length.

**Implemented (AEG-046):** `GET /analytics/noise?from=&to=&limit=` returns top alert fingerprints by volume. `GET /analytics/on-call-load?from=&to=` returns page counts per on-call user (from `paged` / `escalated` timeline events).

**Implemented (AEG-048):** `GET /analytics/overview?from=&to=&compare_previous=` aggregates MTTA, MTTR, noise, on-call load, handoffs, and escalation stats in one response.

**Implemented (AEG-052):** `POST /setup/test-alert` (admin) posts a sample alert through the webhook ingest path and returns `{id, status}`.

## Saved views (Phase 4)

| Method | Path | Description |
|--------|------|-------------|
| GET/POST | `/saved-views` | List/create |
| GET/PATCH/DELETE | `/saved-views/{id}` | CRUD |

**Implemented (AEG-035):** Saved views store a `filter` JSON object (same fields as alert list query params). `shared=true` makes the view visible to all authenticated users. Only the owner can update or delete.

Create body:

```json
{
  "name": "Critical platform",
  "filter": {"severity": "critical", "label_key": "team", "label_value": "platform"},
  "shared": false
}
```

## Webhook payload (alert intake)

Accepts common monitoring shapes. Minimum fields extracted:

```json
{
  "status": "firing",
  "labels": {"alertname": "HighCPU", "team": "platform"},
  "annotations": {"summary": "CPU high on host-1"},
  "startsAt": "2026-06-26T12:00:00Z"
}
```

Response `202`:

```json
{"id": "uuid", "status": "accepted"}
```

## References

- Architecture: [`02-architecture.md`](./02-architecture.md)
- Security: [`09-security.md`](./09-security.md)
- Localization: [`11-localization.md`](./11-localization.md)
