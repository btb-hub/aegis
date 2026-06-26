CREATE TABLE schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id UUID NOT NULL REFERENCES teams (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    timezone TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (team_id, name)
);

CREATE TABLE schedule_layers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    schedule_id UUID NOT NULL REFERENCES schedules (id) ON DELETE CASCADE,
    priority INT NOT NULL DEFAULT 0,
    rotation_type TEXT NOT NULL DEFAULT 'weekly' CHECK (rotation_type IN ('weekly')),
    handoff_weekday INT NOT NULL CHECK (handoff_weekday BETWEEN 0 AND 6),
    handoff_time TIME NOT NULL,
    participant_user_ids UUID[] NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX schedules_team_id_idx ON schedules (team_id);
CREATE INDEX schedule_layers_schedule_id_idx ON schedule_layers (schedule_id);
