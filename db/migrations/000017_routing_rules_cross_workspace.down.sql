DROP INDEX IF EXISTS routing_rules_workspace_idx;

ALTER TABLE routing_rules
    DROP COLUMN IF EXISTS cross_workspace,
    DROP COLUMN IF EXISTS workspace_id;
