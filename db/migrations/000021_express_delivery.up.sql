ALTER TABLE notifications ADD COLUMN delivery_key TEXT NOT NULL DEFAULT 'legacy';
ALTER TABLE notifications ADD COLUMN delivery_error TEXT;
ALTER TABLE notifications ADD COLUMN failure_permanent BOOLEAN NOT NULL DEFAULT false;
CREATE INDEX notifications_delivery_idx ON notifications(incident_id, integration_id, delivery_key, status);
CREATE TABLE express_command_receipts (
 integration_id UUID NOT NULL REFERENCES integrations(id) ON DELETE CASCADE,
 command_id TEXT NOT NULL,
 outcome TEXT NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 PRIMARY KEY(integration_id,command_id)
);
CREATE TABLE oncall_deliveries (
 team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
 provider TEXT NOT NULL,
 destination TEXT NOT NULL,
 fingerprint TEXT NOT NULL,
 config_version TEXT NOT NULL,
 publication_key TEXT NOT NULL DEFAULT '',
 external_ref TEXT,
 status TEXT NOT NULL CHECK(status IN ('pending','sent','failed')),
 attempts INT NOT NULL DEFAULT 0,
 retry_at TIMESTAMPTZ,
 last_error TEXT,
 PRIMARY KEY(team_id,provider,destination)
);
ALTER TABLE jobs ADD COLUMN dedup_key TEXT;
CREATE UNIQUE INDEX jobs_dedup_idx ON jobs(dedup_key) WHERE dedup_key IS NOT NULL;

CREATE TABLE express_notification_results (
 integration_id UUID NOT NULL REFERENCES integrations(id) ON DELETE CASCADE,
 sync_id TEXT NOT NULL,
 status TEXT NOT NULL,
 reason TEXT NOT NULL DEFAULT '',
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 PRIMARY KEY(integration_id,sync_id)
);
CREATE TABLE express_outbound (
 integration_id UUID NOT NULL REFERENCES integrations(id) ON DELETE CASCADE,
 delivery_key TEXT NOT NULL,
 team_id UUID REFERENCES teams(id) ON DELETE CASCADE,
 destination TEXT,
 publication_key TEXT,
 sync_id TEXT NOT NULL,
 status TEXT NOT NULL DEFAULT 'sent',
 delivery_error TEXT,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 PRIMARY KEY(integration_id,delivery_key)
);
CREATE INDEX express_outbound_sync_idx ON express_outbound(integration_id,sync_id);
CREATE INDEX jobs_running_lease_idx ON jobs(updated_at) WHERE status='running';
