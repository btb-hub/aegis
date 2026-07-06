CREATE TABLE escalation_paths (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    from_team_id UUID NOT NULL REFERENCES teams (id) ON DELETE CASCADE,
    to_team_id UUID NOT NULL REFERENCES teams (id) ON DELETE CASCADE,
    workspace_id UUID NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
    cross_workspace BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (from_team_id, to_team_id)
);

CREATE INDEX escalation_paths_from_team_idx ON escalation_paths (from_team_id);
CREATE INDEX escalation_paths_to_team_idx ON escalation_paths (to_team_id);
CREATE INDEX escalation_paths_workspace_idx ON escalation_paths (workspace_id);
