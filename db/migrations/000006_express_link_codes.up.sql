CREATE TABLE express_link_codes (
    code TEXT PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX express_link_codes_expires_at_idx ON express_link_codes (expires_at);
