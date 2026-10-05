# Incident integration release acceptance

Status: implementation verified locally; **unreleased**. Deployed Jira/eXpress acceptance and the release version remain pending. Do not restore production credentials or announce a version based on fixture results.

## Local verification (2026-10-05)

- `make lint type test`: passed, including localization and required coverage.
- Isolated PostgreSQL regressions: comment/resolution rollback, durable command/reply deduplication, early/duplicate notification callbacks, concurrent daily scheduling and permanent-failure suppression, reversible migrations, expired-job recovery/fencing, and delayed rotation callbacks: passed.
- Existing alert simulator through the actual API and worker, using isolated PostgreSQL and local Jira Server/DC/BotX fixtures: passed. Verified one routed alert, automatic issue creation, stored URL, initial assignment, independent shared/private notifications, signed Acknowledge and duplicate command acceptance, general/resolution comments mirrored, `/oncall` with MSK shift ranges, manual publication and administrator settings updates.
- Cloud v3/Basic/ADF and Server/DC v2/PAT/text contracts, handoff assignment/origin and uncertain-write reconciliation: regression fixtures passed.
- The simulator bootstrap needed its existing routing payload updated to include `workspace_id`; its regression suite passed.

Fixture acceptance proves application wiring. It does **not** verify an actual Jira board's filters, deployment permissions, BotX result delivery or real personal chats.

## Deploy and accept

Apply migrations in order: `000020_incident_jira`, `000021_express_delivery`, `000022_oncall_settings`. Rollback is supported by corresponding down migrations, after stopping binaries that use the new schema. Downgrading discards the new synchronization/delivery/settings state; preserve data before a production rollback.

Configure Jira deployment explicitly (`server_dc` or `cloud`) and validate authentication with **Test connection**. Keep the existing PAT/Bearer mode for Server/DC. For legacy incident keys without a recorded connector, associate the known original Jira connector/project before enabling synchronization; Aegis will not guess a new destination.

Restore the global eXpress integration in Integrations, with empty workspace scope, its actual bot credentials and `oncall_group_chat_id`. Workspace settings govern private paging. Linked engineers must have opened their personal bot chat. Configure the callback base and notification-result callback per [eXpress integration](integrations/express.md).

Check Settings → daily on-call publication. Initial default is 03:00 UTC (06:00 MSK). Save the deployment's intended local time/timezone; the worker observes it within one minute. Actual shift ranges in announcements and `/oncall` remain in Europe/Moscow independently of publication timezone.

Using a controlled non-production route (or an agreed production acceptance route), generate one alert through the [existing simulator](../devtools/alert-simulator/README.md):

```sh
go run ./devtools/alert-simulator/cmd/alert-simulator -once -team <configured-routing-label>
```

Verify:

1. One Aegis incident, one automatically created Jira issue, real stored issue URL, correct assignee, and visibility on the deployment's on-call board. **Test connection** only checks authentication.
2. Shared channel post plus assigned engineer's private page, each with Acknowledge. Acknowledge from either location; confirm a visible localized response and an acknowledged incident. Duplicate callbacks must not add replies or repeat transitions.
3. Add a general comment, resolve with a note, then comment on the resolved incident. Confirm author/timestamp/line breaks locally and ordered comments in Jira. Check handoff retains the originating Jira connector and updates assignment.
4. Send `/oncall` from an unlinked identity; verify all teams and actual MSK boundaries. Manually publish a team and confirm linked mentions/name fallback.
5. Deliver a notification-result failure and check its saved error. Confirm one destination's failure does not resend another successful message, and permanent announcement failures do not enqueue every minute.

Record the released version, environment, time, test incident/issue links and each outcome. Provide these results to DevOps so they can restore the global eXpress connector. Installation-specific connector/chat IDs belong in configuration, never source code.

## Operational recovery

Integration jobs retry transient failures with bounded backoff. Running jobs have a two-minute lease renewed every 30 seconds; expired claims are recovered and completion is fenced by attempt. Durable destination state suppresses already accepted sends. Jira writes use pre-request attempt markers and label/comment lookup; unresolved uncertain writes require Jira inspection before an operator clears a marker. See [Jira recovery](integrations/jira.md).

Asynchronous incident delivery failures appear on the timeline. Bot reply failures remain in `express_outbound`; publication failures remain in `oncall_deliveries`. A changed connector configuration, changed rotation, next daily date or manual publication permits a new announcement attempt.
