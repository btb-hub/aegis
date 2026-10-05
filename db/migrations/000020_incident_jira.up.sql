ALTER TABLE incidents ADD COLUMN jira_integration_id UUID REFERENCES integrations(id) ON DELETE RESTRICT;
ALTER TABLE incidents ADD COLUMN jira_issue_url TEXT;
CREATE TABLE jira_sync_state (
 incident_id UUID PRIMARY KEY REFERENCES incidents(id) ON DELETE CASCADE,
 integration_id UUID REFERENCES integrations(id) ON DELETE RESTRICT,
 config JSONB NOT NULL DEFAULT '{}',
 issue_key TEXT,
 create_attempted BOOLEAN NOT NULL DEFAULT false,
 assignee_id UUID REFERENCES users(id) ON DELETE SET NULL,
 last_error TEXT
);
CREATE TABLE jira_comment_sync (
 event_id UUID PRIMARY KEY REFERENCES timeline_events(id) ON DELETE CASCADE,
 incident_id UUID NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
 external_id TEXT,
 attempted BOOLEAN NOT NULL DEFAULT false,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
