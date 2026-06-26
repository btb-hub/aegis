# Architecture

## Overview

Aegis is a three-service monorepo plus PostgreSQL. The API stays fast on the request path; the worker
does all heavy lifting. There is **no Redis** — async work uses a `jobs` table in Postgres.

```text
monitoring  ->  [ API (Go/Gin) ]  ->  validate + store alert + insert job  ->  202
                        |
                        v
                   [ Postgres ]
                        |
                        v
                 [ Worker (Go) ]  ->  dedup / route / incident / Jira / page / escalate
```

## Components

| Component | Path | Responsibility |
|-----------|------|----------------|
| API | `apps/api` | HTTP: OIDC auth, CRUD, alert webhook, connector callbacks, analytics reads |
| Worker | `apps/worker` | Poll `jobs`, process alerts, connectors, escalations, on-call materialisation |
| Web | `apps/web` | React + TS + Vite, Tailwind (design tokens), TanStack Query, react-i18next (`en`, `ru`), Storybook |
| Database | Postgres 16 | All state + job queue |
| Migrations | `db/migrations/` | golang-migrate SQL |
| Queries | `db/query/` | sqlc-generated Go |
| i18n (Go) | `pkg/i18n/` | Chat page templates keyed by locale |

## Go layout (per service)

```text
apps/api/
  cmd/api/main.go
  internal/handler/
  internal/service/
  internal/repository/
  pkg/
apps/worker/
  cmd/worker/main.go
  internal/...
db/
  migrations/
  query/
```

## Request path (alert intake)

1. `POST /api/v1/alerts/webhook` validates payload and auth (webhook secret).
2. Insert `alerts` row + `jobs` row (`kind=process_alert`) in one transaction.
3. Return `202 Accepted` with alert ID.
4. Worker claims job via `SELECT ... FOR UPDATE SKIP LOCKED`.

## Job runner

- Poll interval configurable (e.g. 1s).
- Claim: `status=pending AND run_at <= now()`.
- On failure: increment `attempts`, set `last_error`, reschedule with exponential backoff.
- Escalation: insert job with `kind=escalate_incident`, `run_at = now() + policy delay`.

## Connector pattern

```go
type TicketProvider interface {
    CreateTicket(ctx context.Context, inc Incident) (TicketRef, error)
    UpdateAssignee(ctx context.Context, ref TicketRef, user User) error
}

type ChatProvider interface {
    SendPage(ctx context.Context, inc Incident, target User) (MessageRef, error)
    ParseAckCallback(r *http.Request) (AckEvent, error)
}
```

Registry loads enabled integrations from DB. Worker invokes providers; API handles inbound callbacks.

## Authentication

OIDC authorization-code flow for **Google**, **Slack**, **eXpress**:

1. `GET /auth/{provider}/login` → redirect to IdP.
2. `GET /auth/{provider}/callback` → exchange code, upsert `users`, create `sessions`, set cookie.
3. Session middleware on protected routes.

Slack OIDC (sign-in) and Slack bot token (paging) are separate configuration keys.

No local passwords. No self-hosted IdP.

## Design system

Web UI follows [`12-design-system.md`](./12-design-system.md); visual reference
[`design_system.html`](./design_system.html). Tokens map to Tailwind theme variables; shared
components live under `apps/web/src/components/ui/`. **Storybook** (`apps/web`, port 6006) is the
living catalog of component states; feature UI must reuse components documented there.

## Localization

Web UI strings live in `apps/web/src/locales/{en,ru}/`. Outbound chat copy uses `pkg/i18n` keyed by
the recipient's `users.locale`. API errors expose a stable `code`; the web client maps codes to
localized messages. Full rules: [`11-localization.md`](./11-localization.md).

## Errors

All API errors:

```json
{"code": "VALIDATION_ERROR", "message": "Human-readable", "details": {}}
```

## Observability

- `/healthz` — process up
- `/readyz` — Postgres reachable
- `/metrics` — Prometheus exposition

## References

- Data model: [`03-data-model.md`](./03-data-model.md)
- API: [`04-api-spec.md`](./04-api-spec.md)
- Security: [`09-security.md`](./09-security.md)
- Design system: [`12-design-system.md`](./12-design-system.md)
- Localization: [`11-localization.md`](./11-localization.md)
