CREATE TABLE workspaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO workspaces (id, name, slug, description)
VALUES ('00000000-0000-0000-0000-000000000001', 'Default', 'default', 'Default workspace for existing teams');

ALTER TABLE teams ADD COLUMN workspace_id UUID REFERENCES workspaces (id);

UPDATE teams SET workspace_id = '00000000-0000-0000-0000-000000000001' WHERE workspace_id IS NULL;

ALTER TABLE teams ALTER COLUMN workspace_id SET NOT NULL;

CREATE INDEX teams_workspace_id_idx ON teams (workspace_id);
