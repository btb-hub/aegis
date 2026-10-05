DROP TABLE jira_comment_sync;
DROP TABLE jira_sync_state;
ALTER TABLE incidents DROP COLUMN jira_issue_url;
ALTER TABLE incidents DROP COLUMN jira_integration_id;
