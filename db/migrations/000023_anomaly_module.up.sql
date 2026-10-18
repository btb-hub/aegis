CREATE TABLE anomaly_module_episodes (
    installation_id TEXT NOT NULL,
    episode_id UUID NOT NULL,
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    metric_id UUID NOT NULL,
    series_id TEXT NOT NULL CHECK (series_id ~ '^[0-9a-f]{64}$'),
    sequence BIGINT NOT NULL CHECK (sequence > 0),
    state TEXT NOT NULL CHECK (state IN ('firing','resolved')),
    alert_id UUID NOT NULL REFERENCES alerts(id),
    occurred_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (installation_id, episode_id)
);
CREATE TABLE anomaly_module_receipts (
    installation_id TEXT NOT NULL,
    event_id UUID NOT NULL,
    payload_hash TEXT NOT NULL,
    alert_id UUID NOT NULL REFERENCES alerts(id),
    ignored BOOLEAN NOT NULL DEFAULT false,
    received_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (installation_id,event_id)
);
CREATE INDEX anomaly_module_episodes_alert_idx ON anomaly_module_episodes(alert_id);
