# Integrations

Aegis talks to external systems through a small set of connectors. All outbound calls run in the
**worker** with retry, idempotency keys, and recorded fixtures in tests.

## Connector types

| Type | Interface | MVP providers |
|------|-----------|---------------|
| Ticket | `TicketProvider` | Jira |
| Chat | `ChatProvider` | Slack, eXpress |

## Registry

- `integrations` table stores `kind` + JSON `config`.
- Worker loads enabled rows at job start; missing provider skipped gracefully.

## Contract

### TicketProvider

- `CreateTicket(ctx, incident) → (external_key, error)`
- `UpdateAssignee(ctx, key, user) → error`
- Optional inbound webhook for status sync (Jira).

### ChatProvider

- `SendPage(ctx, incident, user) → (message_ref, error)` — message text uses recipient `users.locale` via `pkg/i18n`
- `ParseAckCallback(request) → (incident_id, actor, error)`
- `TestConnection(ctx) → error`

## Failure handling (REQ-INT-06)

- Errors logged; `notifications.status = failed`.
- Retry via job `attempts` with backoff.
- One connector down does not block others.

## Admin configuration

**Preferred UI:** `/integrations` — create, edit credentials, Test connection, enable/disable, delete
each connector independently. Admins must not need the setup wizard to configure Jira, Slack, or
eXpress. See [`EPIC-13`](../../backlog/epics/EPIC-13-integration-admin.md).

Config JSON fields per provider: [`jira.md`](./jira.md), [`slack.md`](./slack.md),
[`express.md`](./express.md).

## Test connection (REQ-INT-05)

- `POST /integrations/{id}/test` invokes provider's `TestConnection`.
- **Integrations** page surfaces pass/fail with an actionable error message.
- The setup wizard may still call the same API, but it is not the primary configure path.

## Specs

- [`jira.md`](./jira.md)
- [`slack.md`](./slack.md)
- [`express.md`](./express.md)
- Localization: [`../11-localization.md`](../11-localization.md)

## Epics

- Connector spine: [`EPIC-03-integrations`](../../backlog/epics/EPIC-03-integrations.md)
- Independent admin configure UI: [`EPIC-13-integration-admin`](../../backlog/epics/EPIC-13-integration-admin.md)
