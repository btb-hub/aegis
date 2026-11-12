-- INC-NUM-DB: assign stable global numbers to old incidents in (created_at, id)
-- order and allocate unique increasing numbers for future concurrent inserts.
-- Gaps after rollback/deletion are allowed; UUIDs and relations stay unchanged.
-- Evidence: pkg/db/incident_number_test.go::TestIncidentNumberMigrationAndStore.
-- The migration runner executes this file in one transaction. Block inserts
-- until backfill and the default are both installed.
LOCK TABLE incidents IN ACCESS EXCLUSIVE MODE;
CREATE SEQUENCE incident_number_seq;
ALTER TABLE incidents ADD COLUMN number BIGINT;
WITH numbered AS (
    SELECT id, row_number() OVER (ORDER BY created_at, id) AS number
    FROM incidents
)
UPDATE incidents i SET number = n.number FROM numbered n WHERE n.id = i.id;
SELECT setval('incident_number_seq', COALESCE(max(number), 0) + 1, false) FROM incidents;
ALTER SEQUENCE incident_number_seq OWNED BY incidents.number;
ALTER TABLE incidents
    ALTER COLUMN number SET DEFAULT nextval('incident_number_seq'),
    ALTER COLUMN number SET NOT NULL,
    ADD CONSTRAINT incidents_number_unique UNIQUE (number),
    ADD CONSTRAINT incidents_number_positive CHECK (number > 0);
