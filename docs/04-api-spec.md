# API specification

Base URL: `/api/v1`. JSON request/response unless noted. Errors: `{code, message, details}`.

OpenAPI schema generated from code in `apps/api` (future story).

## Auth

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/auth/{provider}/login` | — | Redirect to OIDC (`google`, `slack`, `express`) |
| GET | `/auth/{provider}/callback` | — | OIDC callback; sets session cookie |
| POST | `/auth/logout` | session | Invalidate session |
| GET | `/auth/me` | session | Current user + role + locale |
| PATCH | `/auth/me` | session | Update profile fields (`locale`: `en` \| `ru`) |

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
| GET | `/alerts` | session | List with filters, pagination |
| GET | `/alerts/{id}` | session | Detail |
| GET | `/alerts/export` | session | CSV stream |

Query params for list: `severity`, `status`, `team_id`, `from`, `to`, `q` (search), `group_by`, `page`, `page_size`.

## Teams & shifts (Phase 1)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/teams` | session | List teams |
| POST | `/teams` | session + admin | Create team |
| GET | `/teams/{id}` | session | Team detail |
| PATCH | `/teams/{id}` | session + admin | Update team |
| DELETE | `/teams/{id}` | session + admin | Delete team |
| GET | `/teams/{id}/members` | session | List memberships |
| POST | `/teams/{id}/members` | session + admin | Add member |
| PATCH | `/teams/{id}/members/{userId}` | session + admin | Update member `team_role` |
| DELETE | `/teams/{id}/members/{userId}` | session + admin | Remove member |

Create team body: `{"name": "Platform", "description": "optional"}`.

Add member body: `{"user_id": "uuid", "team_role": "member" | "lead"}` (defaults to `member`).

Member response includes `user_id`, `team_role`, `email`, `display_name`.

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

Schedule and override changes enqueue a `materialise_oncall` worker job for the team. The worker also runs a nightly job for all teams with schedules.

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
| POST | `/incidents/{id}/bounce` | Bounce to L2 `{note}` |

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

## Saved views (Phase 4)

| Method | Path | Description |
|--------|------|-------------|
| GET/POST | `/saved-views` | List/create |
| GET/PATCH/DELETE | `/saved-views/{id}` | CRUD |

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
