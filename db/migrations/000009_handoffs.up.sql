CREATE TABLE handoffs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    incident_id UUID NOT NULL REFERENCES incidents (id) ON DELETE CASCADE,
    from_user_id UUID REFERENCES users (id) ON DELETE SET NULL,
    to_user_id UUID REFERENCES users (id) ON DELETE SET NULL,
    from_team_id UUID NOT NULL REFERENCES teams (id) ON DELETE CASCADE,
    to_team_id UUID NOT NULL REFERENCES teams (id) ON DELETE CASCADE,
    reason TEXT,
    bounced_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX handoffs_incident_created_idx ON handoffs (incident_id, created_at DESC);
