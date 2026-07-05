ALTER TABLE teams
    ADD COLUMN support_tier TEXT CHECK (support_tier IS NULL OR support_tier IN ('l1', 'l2', 'l3', 'noc'));

CREATE INDEX teams_support_tier_idx ON teams (support_tier) WHERE support_tier IS NOT NULL;
