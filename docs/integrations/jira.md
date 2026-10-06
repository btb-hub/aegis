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

## Acceptance

Create a routed incident to verify issue creation, assignment and board visibility. Test connection checks authentication only.
