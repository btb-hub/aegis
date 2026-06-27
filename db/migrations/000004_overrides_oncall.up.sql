CREATE TABLE overrides (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id UUID NOT NULL REFERENCES teams (id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    start_at TIMESTAMPTZ NOT NULL,
    end_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (end_at > start_at)
);

CREATE TABLE on_call_slots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id UUID NOT NULL REFERENCES teams (id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    start_at TIMESTAMPTZ NOT NULL,
    end_at TIMESTAMPTZ NOT NULL,
    source TEXT NOT NULL DEFAULT 'rotation' CHECK (source IN ('rotation', 'override')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (end_at > start_at)
);

CREATE INDEX overrides_team_id_idx ON overrides (team_id);
CREATE INDEX overrides_team_time_idx ON overrides (team_id, start_at, end_at);
CREATE INDEX on_call_slots_team_time_idx ON on_call_slots (team_id, start_at, end_at);
