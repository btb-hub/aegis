# Integration: Jira

Ticket provider. Implements `TicketProvider`.

## Config (env / integration JSON)

- `base_url` — Jira Cloud or Server URL
- `email` + `api_token` — basic auth (Cloud) or PAT
- `project_key` — default project for new issues
- `issue_type` — default `Task` or `Incident`

## Create ticket

On incident open, worker calls Jira REST API:

- Summary: incident title
- Description: link back to Aegis incident, alert labels
- Labels: `aegis`, team slug

Store `jira_issue_key` on incident.

## Assignee sync

- On handoff or reassignment, `PUT` issue assignee if mappable Jira account exists.

## Inbound (optional MVP+)

- Jira webhook on status → resolved maps to incident resolve (story-gated).

## Test connection

- `GET /rest/api/3/myself` or `/rest/api/2/myself`.

## Tests

- Recorded JSON fixtures in `apps/worker/testdata/jira/` — no live Jira in CI.

## References

- REQ-INT-02
