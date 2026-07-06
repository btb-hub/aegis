DROP INDEX IF EXISTS integrations_workspace_id_idx;
DROP INDEX IF EXISTS integrations_workspace_kind_idx;
DROP INDEX IF EXISTS integrations_global_kind_idx;

ALTER TABLE integrations DROP COLUMN IF EXISTS workspace_id;

ALTER TABLE integrations ADD CONSTRAINT integrations_kind_key UNIQUE (kind);
