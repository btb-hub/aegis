# Aegis overview

Map of the current product and its specifications, refreshed through November 13, 2026
(main `c604e55`). Styled HTML for a docs host: [`overview.html`](./overview.html).

Aegis takes a firing alert, turns it into one owned incident, opens a Jira ticket, and pages the
person who is actually on call through Slack and eXpress. Sign in with a configured Google,
Slack, or eXpress OIDC provider.
The web UI and chat pages ship in English and Russian.

## North stars

| Easy to | Means |
|---------|--------|
| **Set up** | Bootstrap a fresh installation, then configure sign-in, bots and behavior in Settings. Existing installations import their environment configuration before upgrading. |
| **Use** | One screen per job: who's on call, the alert workspace, the incident timeline. Ack from chat. L2 hands off to L3 in one click. |
| **Analyse** | Five in-product questions: are we faster, what's noisy, is load fair, how's L2 to L3, how are we escalating. |

## Four MVP features

| Feature | Spec | Epic |
|---------|------|------|
| Shifts calendar | [`features/shifts-calendar.md`](./features/shifts-calendar.md) | [EPIC-02](../backlog/epics/EPIC-02-shifts.md) |
| Incident management | [`features/incident-management.md`](./features/incident-management.md) | [EPIC-04](../backlog/epics/EPIC-04-incidents.md) |
| Alerting workspace | [`features/alerting.md`](./features/alerting.md) | [EPIC-05](../backlog/epics/EPIC-05-alerting.md) |
| L2 to L3 transparency | [`features/l2-l3-transparency.md`](./features/l2-l3-transparency.md) | [EPIC-06](../backlog/epics/EPIC-06-l2-l3.md) |

## Architecture at a glance

The request path stays fast: validate, store, enqueue, return. Dedup, routing, tickets, paging, and
escalation run in the worker.

```
monitoring  ->  [ API (Go) ]  ->  validate + store alert + insert job  ->  202
                     |
                     v
                 [ Postgres ]
                     |
                     v
              [ Worker (Go) ]  ->  dedup / group  ->  route to team  ->  create incident
                                      |                                     |
                                      v                                     v
                               assign on-call                       create Jira ticket
                                      |                                     |
                                      +------------------+------------------+
                                                         v
                                              fan-out: Slack / eXpress
                                                         |
                                                  ack  <-+->  escalate (timer)
```

| Piece | Role |
|-------|------|
| `apps/api` | Go/Gin. OIDC, CRUD, alert intake, callbacks, analytics reads. |
| `apps/worker` | Postgres job poller. Dedup, routing, incident + ticket, paging, escalation, on-call materialisation. |
| `apps/web` | React + TypeScript + Vite. Settings, calendar, alert workspace, responsive incident view, dashboards. |
| data | PostgreSQL 16 only, including settings and durable delivery queues. golang-migrate; Compose for local builds, GHCR image with external Postgres for production. |

Full detail: [`02-architecture.md`](./02-architecture.md), [`03-data-model.md`](./03-data-model.md),
[`04-api-spec.md`](./04-api-spec.md).

## Integrations

Each connector implements a shared interface. Any subset works; a failing one degrades. Incident
delivery runs in the worker; connection tests and Account authorization run in the API. Shared
messages and acknowledgement feedback have bounded retries; ordinary failed jobs remain failed.

| Kind | Connector | Spec |
|------|-----------|------|
| Ticket | Jira | [`integrations/jira.md`](./integrations/jira.md) |
| Chat | Slack | [`integrations/slack.md`](./integrations/slack.md) |
| Chat | eXpress | [`integrations/express.md`](./integrations/express.md) |

Hub: [`integrations/README.md`](./integrations/README.md). Auth: [`09-security.md`](./09-security.md).
Stories: [EPIC-03](../backlog/epics/EPIC-03-integrations.md).

## Roadmap

The original MVP phases 0–7 are complete. Later changes add workspace administration, Account
paging connections, shared notifications, Settings and the optional anomaly module. Phases and
exit criteria: [`backlog/roadmap.md`](../backlog/roadmap.md).

## Changes since the October 7 documentation refresh

| Area | Current behavior | Details |
| --- | --- | --- |
| Configuration | PostgreSQL-backed Settings replaces the wizard; `/setup` and `/integrations` redirect there. Fresh installs use `/bootstrap`; upgrades need an explicit configuration import. | [Settings](features/settings.md), [migration](configuration-migration.md) |
| Anomalies | Optional private detector behind Aegis authentication. Recovery resolves its alert and leaves incident closure to responders. A local Mimir sandbox is available. | [Module](features/anomaly-detection.md), [sandbox](../devtools/anomaly-sandbox/README.md) |
| Incident UI | Permanent labels such as `INC-01`; responsive queue/detail navigation; URL status filters and selection retained after actions. | [Incident guide](how-to/incidents.md) |
| Jira | Detail links use the effective workspace Jira URL, including context paths. New tickets attempt assignee synchronization; Server/DC has an exact username fallback. | [Jira](integrations/jira.md), [API](04-api-spec.md) |
| eXpress | Account offers bot `/link` codes without SSO. Successful acknowledgements in the configured group confirm through the shared post; personal chats and other outcomes retain feedback. Blank feedback sends nothing. | [Connections](features/paging-connections.md), [delivery](features/incident-chat-notifications.md) |

## How the agent builds it

Pick the top ready story, ship a small PR, loop. Contract: [`CLAUDE.md`](../CLAUDE.md),
[`10-agent-loop.md`](./10-agent-loop.md).

## Documentation set

| Doc | What it is |
|-----|------------|
| [`README.md`](../README.md) | Entry point + doc map |
| [`how-to/README.md`](./how-to/README.md) | User manual for the on-call workflow |
| [`CLAUDE.md`](../CLAUDE.md) | Agent working contract |
| [`00-product-brief.md`](./00-product-brief.md) | Vision and scope |
| [`01-prd.md`](./01-prd.md) | Numbered requirements |
| [`12-design-system.md`](./12-design-system.md) | UI tokens and components |
| [`user-guide.md`](./user-guide.md) | Day-to-day use by role |
| [`backlog/epics/`](../backlog/epics/) | Epics and stories |
