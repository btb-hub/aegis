-- Model A: routing rule ownership is the configuring workspace, not the target team's home.
ALTER TABLE routing_rules
    ADD COLUMN workspace_id UUID REFERENCES workspaces (id) ON DELETE CASCADE,
    ADD COLUMN cross_workspace BOOLEAN NOT NULL DEFAULT false;

-- Backfill ownership from the target team's home workspace.
UPDATE routing_rules rr
SET workspace_id = t.workspace_id
FROM teams t
WHERE t.id = rr.team_id
  AND rr.workspace_id IS NULL;

ALTER TABLE routing_rules
    ALTER COLUMN workspace_id SET NOT NULL;

CREATE INDEX routing_rules_workspace_idx ON routing_rules (workspace_id);
