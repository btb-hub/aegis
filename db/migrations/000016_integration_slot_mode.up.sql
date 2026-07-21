-- 000016_integration_slot_mode.up.sql
ALTER TABLE integrations
  ADD COLUMN mode TEXT;

ALTER TABLE integrations
  ADD CONSTRAINT integrations_mode_check
  CHECK (mode IS NULL OR mode IN ('inherit', 'custom'));

-- Existing workspace rows become inherit (typical project_key overrides).
UPDATE integrations
SET mode = 'inherit'
WHERE workspace_id IS NOT NULL AND mode IS NULL;

-- Ensure three slots per workspace.
INSERT INTO integrations (kind, name, config, enabled, workspace_id, mode)
SELECT k.kind, k.kind, '{}'::jsonb, true, w.id, 'inherit'
FROM workspaces w
CROSS JOIN (VALUES ('jira'), ('slack'), ('express')) AS k(kind)
WHERE NOT EXISTS (
  SELECT 1 FROM integrations i
  WHERE i.workspace_id = w.id AND i.kind = k.kind
);
