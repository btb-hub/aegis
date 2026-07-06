ALTER TABLE integrations ADD COLUMN workspace_id UUID REFERENCES workspaces (id) ON DELETE CASCADE;

ALTER TABLE integrations DROP CONSTRAINT IF EXISTS integrations_kind_key;

CREATE UNIQUE INDEX integrations_global_kind_idx ON integrations (kind) WHERE workspace_id IS NULL;

CREATE UNIQUE INDEX integrations_workspace_kind_idx ON integrations (workspace_id, kind) WHERE workspace_id IS NOT NULL;

CREATE INDEX integrations_workspace_id_idx ON integrations (workspace_id) WHERE workspace_id IS NOT NULL;
