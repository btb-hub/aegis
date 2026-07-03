ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_url TEXT;

CREATE TABLE user_identities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    provider TEXT NOT NULL CHECK (provider IN ('google', 'slack', 'express', 'dev')),
    provider_sub TEXT NOT NULL,
    linked_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (provider, provider_sub),
    UNIQUE (user_id, provider)
);

CREATE INDEX user_identities_user_id_idx ON user_identities (user_id);

INSERT INTO user_identities (user_id, provider, provider_sub, linked_at)
SELECT id, provider, provider_sub, created_at
FROM users
ON CONFLICT (provider, provider_sub) DO NOTHING;

CREATE TABLE IF NOT EXISTS audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id UUID REFERENCES users (id) ON DELETE SET NULL,
    action TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id UUID,
    details JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX audit_log_actor_id_idx ON audit_log (actor_id);
CREATE INDEX audit_log_created_at_idx ON audit_log (created_at DESC);
