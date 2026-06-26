# Feature: Incident management

Implements PRD §2 (`REQ-INC-*`).

## Problem

Alerts fire everywhere; incidents are tracked manually; Jira tickets are copy-pasted; pages get lost.

## Solution

Webhook → store → async process → dedup → route → incident → Jira → page → ack → optional escalate.

## Lifecycle

```text
firing alert -> (dedup) -> open incident -> page on-call
                    |-> acknowledged -> resolved
                    |-> escalation timer -> re-page / next step
```

States: `open`, `acknowledged`, `resolved`. Transitions logged to `timeline_events`.

## Dedup

- Fingerprint from stable label subset (configurable default: `alertname` + `instance`).
- Firing alerts with same fingerprint within window attach to existing open incident.

## Routing

- Rules evaluated by priority; first `match_labels` wins.
- Default rule sends to `team` label if present.

## Paging

- Worker fans out to enabled chat integrations for assignee.
- Partial failure OK: record per-connector status in `notifications`.

## Escalation

- Policy: unacked after N minutes → job `escalate_incident` at `run_at`.
- MVP: one step (re-page assignee or notify team channel).

## UI

- Incident list with status filters.
- Detail: timeline, linked alerts, Jira link, ack/resolve actions.

## References

- Epic: [`EPIC-04-incidents`](../../backlog/epics/EPIC-04-incidents.md)
- Integrations: [`../integrations/`](../integrations/)
