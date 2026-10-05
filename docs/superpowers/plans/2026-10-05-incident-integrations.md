# Incident and integration improvements

User-approved scope: Jira Server/DC v2 and Cloud v3 compatibility; incident timeline comments and optional atomic resolution notes synced asynchronously to the original Jira connector; independent global-channel and assigned-engineer eXpress pages; authenticated 202 command responses with durable visible feedback and /oncall; global daily publication time/timezone settings (03:00 UTC default), MSK shift ranges and bounded publication retries.

Implementation groups: Jira protocol, comment/sync persistence, eXpress delivery/commands, publication settings, UI and acceptance verification. Preserve /link, Acknowledge, linked mentions, rotation/manual announcements and workspace paging resolution. Ship reversible migrations, en/ru strings and API docs. Run make lint type test and recorded integration fixtures. Live simulator acceptance requires deployed Jira/eXpress access; no credentials have been supplied.
