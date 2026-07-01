-- Backfill search_tsv for rows ingested before explicit tsvector population.
UPDATE alerts
SET search_tsv = to_tsvector('english', coalesce(title, '') || ' ' || coalesce(body, ''))
WHERE search_tsv IS NULL;

-- Default list ordering index (NFR-2 / alerting workspace).
CREATE INDEX IF NOT EXISTS alerts_received_at_idx ON alerts (received_at DESC);
