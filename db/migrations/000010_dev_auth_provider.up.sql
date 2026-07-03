ALTER TABLE users DROP CONSTRAINT IF EXISTS users_provider_check;
ALTER TABLE users ADD CONSTRAINT users_provider_check
    CHECK (provider IN ('google', 'slack', 'express', 'dev'));
