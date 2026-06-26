# Feature: Shifts calendar

Implements PRD §1 (`REQ-SHIFT-*`).

## Problem

On-call rotations live in spreadsheets. Overrides are missed. DST breaks "who's on call now".

## Solution

Teams define schedules in Aegis. The worker materialises `on_call_slots`. The UI and API answer
"who's on call now" instantly and correctly.

## Behaviour

### Schedules

- One team may have multiple schedules (e.g. primary + backup); resolution picks active layer by priority.
- Rotation types: weekly handoff at configured weekday/time in team timezone.
- Participants are ordered; handoff advances the pointer.

### Overrides

- Admin sets override: user X covers slot from T1 to T2.
- Override wins over rotation for overlapping window.

### DST

- All schedule math uses IANA timezone from `schedules.timezone`.
- Materialisation uses `time.Local` in that zone; unit tests cover spring-forward and fall-back.

### Materialisation

- Job `materialise_oncall` runs on schedule CRUD and nightly.
- Generates slots for horizon (e.g. 90 days).

## UI

- Calendar month view with colour per user.
- "On call now" banner on team page.
- Override create dialog with datetime pickers in team timezone.

## API

See [`04-api-spec.md`](../04-api-spec.md) — Teams & shifts.

## References

- Epic: [`EPIC-02-shifts`](../../backlog/epics/EPIC-02-shifts.md)
- Data model: [`03-data-model.md`](../03-data-model.md)
