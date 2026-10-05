# Integration: Jira

## Configuration and compatibility

Set `deployment` to `server_dc` or `cloud`, independently of `auth_type` (`bearer` or `basic`). Existing configurations infer Server/DC from Bearer and Cloud from Basic. An explicit deployment takes precedence. Server/DC PAT authentication remains `Authorization: Bearer`; Cloud normally uses Basic `email:api_token`.

Other fields: `base_url` (including any Jira context path), `project_key`, `api_token`, `email` (required for Basic), and `issue_type` (default `Task`). Workspace slots inherit the global connector or supply custom credentials. Test connection checks authentication, not issue creation or board filters.

| Operation | Server/DC | Cloud |
| --- | --- | --- |
| Authentication check | `GET /rest/api/2/myself` | `GET /rest/api/3/myself` |
| Create issue | `POST /rest/api/2/issue`, string description | `POST /rest/api/3/issue`, ADF description |
| Find assignee | `/rest/api/2/user/search?username=…`, returned `name` | `/rest/api/3/user/search?query=…`, returned `accountId` |
| Assign | Issue update with `fields.assignee.name` | Issue update with `fields.assignee.accountId` |
| Comment | v2 issue comment, string body | v3 issue comment, ADF body |
| Reconcile creation | v2 JQL `/search` | v3 enhanced JQL `/search/jql` |

Search parameters are URL-encoded. Multiple matches require a unique exact username/email match; missing or ambiguous active users are reported. Login redirects are rejected with an actionable status error, and credentials are excluded from errors.

## Incident synchronization

A durable `sync_jira` job is queued when an incident is created, including if it resolves before processing. Issues carry the immutable label `aegis-incident-{incident UUID}` and an Aegis link. The actual issue URL and originating connector are saved on the incident; the UI uses that URL. Handoff continues using that connector and its original Jira/project, with current credentials. Connectors referenced by tickets cannot be deleted; disable or repair them instead.

Initial assignment and handoff assignment are synchronized. Timeline comments are append-only, include author/timestamp, and synchronize in order. Each remote comment contains an immutable `Aegis event: {timeline event UUID}` marker. Local resolution and comments persist if Jira is unavailable. This integration does not change Jira workflow/status.

Attempt markers commit before external writes. On retries, search the issue label or comment marker before sending again. An uncertain write whose result is still absent from Jira remains pending and is never blindly repeated. Retries are bounded (1, 5, 15 minutes); failed jobs and timeline errors remain visible.

For a terminal uncertain write, an administrator must inspect Jira, its project/issue permissions and indexing before retrying. If the remote object exists, requeue `sync_jira` for reconciliation. Only after confirming that it does not exist should its `create_attempted` or comment `attempted` marker be reset and the job requeued. Preserve the original connector and project. Incidents created before this migration with a Jira key but no originating connector require explicit origin association in `jira_sync_state`; they are not automatically routed to a guessed Jira instance.

## Acceptance and references

Use the routed incident simulator to verify creation, stored URL, assignment and visibility in the configured on-call board. A successful authentication check alone cannot verify board membership. See [release acceptance](../incident-integration-release.md).

[Server/DC REST](https://developer.atlassian.com/server/jira/platform/rest/v11003/api-group-user), [Cloud REST](https://developer.atlassian.com/cloud/jira/platform/rest/v3/api-group-issues).
