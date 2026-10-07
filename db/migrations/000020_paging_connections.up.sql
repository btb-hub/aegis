CREATE TABLE user_paging_settings (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider TEXT NOT NULL CHECK (provider IN ('slack', 'express')),
    PRIMARY KEY (user_id, provider)
);

CREATE TABLE paging_authorizations (
    state_hash TEXT PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_hash TEXT NOT NULL REFERENCES sessions(token_hash) ON DELETE CASCADE,
    provider TEXT NOT NULL CHECK (provider IN ('slack', 'express')),
    nonce TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    exchanging BOOLEAN NOT NULL DEFAULT false,
    UNIQUE (user_id, provider)
);
CREATE INDEX paging_authorizations_expires_at_idx ON paging_authorizations(expires_at);
