# Feature: L2↔L3 transparency

Implements PRD §4 (`REQ-L2L3-*`).

## Problem

L2 escalates informally; L3 lacks context; handoff time is invisible to management.

## Solution

One-click handoff to L3 team's current on-call with shared timeline and measurable response time.

## Handoff action

1. L2 clicks **Hand off to L3** on incident.
2. Picks target L3 team (or preconfigured mapping).
3. System:
   - Records `handoffs` row
   - Reassigns `incidents.assignee_id` to L3 on-call
   - Appends timeline event
   - Pages L3 via chat connectors
   - Updates Jira assignee if integration configured

## Shared timeline

- L2 and L3 users with access see identical `timeline_events` — no role-based hiding (REQ-L2L3-03).
- Phase 10 extends this policy to **all support tiers** (L1, NOC included): no tier-based filtering in
  API or UI. See [`support-levels-and-workspaces.md`](./support-levels-and-workspaces.md#timeline-visibility-policy-req-l2l3-03-unchanged).

## Bounce

- L3 **Bounce to L2** with required note.
- Reassigns back to prior L2 owner or team default.

## Analytics

- Time from handoff event to L3 first ack or comment.
- Aggregated in [`08-analytics.md`](../08-analytics.md) handoff widget.

## References

- Epic: [`EPIC-06-l2-l3`](../../backlog/epics/EPIC-06-l2-l3.md)
- Phase 10 (tiers, workspaces, shared timeline policy): [`EPIC-11-support-levels-workspaces`](../../backlog/epics/EPIC-11-support-levels-workspaces.md)
