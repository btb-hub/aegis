-- 000016_integration_slot_mode.down.sql
ALTER TABLE integrations DROP CONSTRAINT IF EXISTS integrations_mode_check;
ALTER TABLE integrations DROP COLUMN IF EXISTS mode;
-- Do not delete auto-created slots on down (data-preserving); only drop mode.
