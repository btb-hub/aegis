-- AEG-111: on-call review of one backup-checker run. A review belongs to a check run,
-- never to the backup, so every new run waits for its own decision.
CREATE TABLE backup_check_reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    check_id TEXT NOT NULL UNIQUE CHECK (length(check_id) BETWEEN 1 AND 128),
    backup TEXT NOT NULL CHECK (length(backup) BETWEEN 1 AND 128),
    decision TEXT NOT NULL CHECK (decision IN ('confirmed', 'rejected')),
    comment TEXT NOT NULL DEFAULT '' CHECK (length(comment) <= 500),
    reviewer_id UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (decision <> 'rejected' OR length(btrim(comment)) > 0)
);
CREATE INDEX backup_check_reviews_backup_idx ON backup_check_reviews (backup, created_at DESC);
