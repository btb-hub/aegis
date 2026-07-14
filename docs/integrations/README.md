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
- Global rows provide optional deployment-wide credentials.
- Every workspace has stable Jira, Slack, and eXpress slots. Workspace creation provisions all
  three as enabled `inherit` slots; the slot migration backfills existing workspaces.
- The worker resolves each kind for the incident's owning team workspace at job start.

## Workspace slots

Each slot has a first-class mode:

- **Inherit** merges an enabled global connector with supported non-secret workspace overlays.
  Jira supports `project_key`; Slack supports `channel_id`. No workspace secrets remain stored when
  a slot switches to Inherit.
- **Custom** uses a complete workspace provider config and does not mix in global credentials.

Resolution first requires an enabled workspace slot. Custom mode then validates the workspace
config; Inherit mode requires an enabled global connector and merges its config with the slot
overlay. Alert notification, escalation, handoff, and Test connection all use this resolver.

Unavailable providers do not fail alert ingestion or the incident workflow. Runtime skips that
provider, continues with other providers, and appends an `integration_skipped` timeline event with
the connector `kind`, a reason code, and an actionable message. Reason codes are
`slot_disabled`, `slot_missing`, `custom_incomplete`, `no_global`, and `global_disabled`.

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
eXpress. The global inventory and each workspace's Integrations panel expose the same provisioned
slots, modes, and effective status. See
[`EPIC-13`](../../backlog/epics/EPIC-13-integration-admin.md).

Config JSON fields per provider: [`jira.md`](./jira.md), [`slack.md`](./slack.md),
[`express.md`](./express.md).

## Test connection (REQ-INT-05)

- `POST /integrations/{id}/test` resolves the effective workspace/global config, then invokes the
  provider's `TestConnection`.
- Resolve failures return a structured validation error; unlike runtime delivery, Test does not
  report a skipped success.
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
