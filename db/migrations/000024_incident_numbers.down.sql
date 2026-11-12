-- Dropping the owned column also drops its sequence. UUIDs and relations remain.
ALTER TABLE incidents DROP COLUMN number;
