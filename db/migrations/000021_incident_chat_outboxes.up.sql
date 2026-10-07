CREATE TABLE incident_channel_events (
    id UUID PRIMARY KEY,
    sequence BIGSERIAL UNIQUE NOT NULL,
    incident_id UUID NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK (kind IN ('created','acknowledged','resolved','escalated')),
    snapshot JSONB NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);

CREATE TABLE incident_channel_deliveries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES incident_channel_events(id) ON DELETE CASCADE,
    integration_id UUID NOT NULL REFERENCES integrations(id) ON DELETE CASCADE,
    provider TEXT NOT NULL CHECK (provider IN ('slack','express')),
    destination TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','sent','failed')),
    attempts INT NOT NULL DEFAULT 0,
    next_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    leased_until TIMESTAMPTZ,
    external_ref TEXT,
    last_error TEXT,
    UNIQUE(event_id,integration_id,destination)
);
CREATE INDEX incident_channel_events_incident_sequence ON incident_channel_events(incident_id,sequence);
CREATE INDEX incident_channel_deliveries_pending ON incident_channel_deliveries(next_at) WHERE status = 'pending';

CREATE TABLE chat_ack_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider TEXT NOT NULL CHECK (provider IN ('slack','express')),
    dedup_key TEXT NOT NULL,
    incident_id UUID NOT NULL,
    workspace_id UUID,
    integration_id UUID NOT NULL REFERENCES integrations(id) ON DELETE CASCADE,
    user_identity TEXT NOT NULL,
    chat_id TEXT NOT NULL DEFAULT '',
    response_url TEXT NOT NULL DEFAULT '',
    locale TEXT NOT NULL DEFAULT 'en',
    outcome JSONB,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','sent','failed')),
    attempts INT NOT NULL DEFAULT 0,
    next_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    leased_until TIMESTAMPTZ,
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(provider,dedup_key)
);
CREATE INDEX chat_ack_requests_pending ON chat_ack_requests(next_at) WHERE status = 'pending';

-- Called inside the incident transaction. Destinations and event details are
-- captured now, rather than reconstructed from a later incident state.
CREATE FUNCTION queue_incident_channel_event(event_id UUID, incident UUID, event_kind TEXT, actor UUID, at_time TIMESTAMPTZ)
RETURNS void LANGUAGE plpgsql AS $$
DECLARE
    owner teams%ROWTYPE;
BEGIN
    -- Serialize sequence allocation with incident state transitions. A slow
    -- escalation transaction can't appear after a later acknowledgement.
    PERFORM 1 FROM incidents WHERE id=incident FOR UPDATE;
    INSERT INTO incident_channel_events(id,incident_id,kind,snapshot,occurred_at)
    SELECT event_id,i.id,event_kind,jsonb_build_object(
        'incident',to_jsonb(i),'team_name',t.name,
        'slack_user_group_id',COALESCE(t.slack_user_group_id,''),
        'actor_name',COALESCE(a.display_name,''),
        'locale',COALESCE(NULLIF(u.locale,''),'en')),
        at_time
    FROM incidents i JOIN teams t ON t.id=i.team_id
    LEFT JOIN users u ON u.id=i.assignee_id LEFT JOIN users a ON a.id=actor
    WHERE i.id=incident
    ON CONFLICT(id) DO NOTHING;
    IF NOT FOUND THEN RETURN; END IF;
    SELECT t.* INTO owner FROM teams t JOIN incidents i ON i.team_id=t.id WHERE i.id=incident;

    INSERT INTO incident_channel_deliveries(event_id,integration_id,provider,destination)
    SELECT event_id, CASE WHEN s.mode='custom' THEN s.id ELSE g.id END,'slack',btrim(owner.slack_channel_id)
    FROM integrations s LEFT JOIN integrations g ON g.kind='slack' AND g.workspace_id IS NULL
    WHERE s.workspace_id=owner.workspace_id AND s.kind='slack' AND s.enabled
      AND (s.mode='custom' OR (s.mode='inherit' AND g.enabled))
      AND NULLIF(btrim(owner.slack_channel_id),'') IS NOT NULL;

    INSERT INTO incident_channel_deliveries(event_id,integration_id,provider,destination)
    SELECT event_id,g.id,'express',btrim(g.config->>'oncall_group_chat_id')
    FROM integrations g WHERE g.kind='express' AND g.workspace_id IS NULL AND g.enabled
      AND NULLIF(btrim(g.config->>'oncall_group_chat_id'),'') IS NOT NULL;
END $$;

CREATE FUNCTION incident_channel_timeline_trigger() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.kind IN ('created','acknowledged','resolved') THEN
        PERFORM queue_incident_channel_event(NEW.id,NEW.incident_id,NEW.kind,NEW.actor_id,clock_timestamp());
    END IF;
    RETURN NEW;
END $$;
CREATE TRIGGER incident_channel_timeline AFTER INSERT ON timeline_events
FOR EACH ROW EXECUTE FUNCTION incident_channel_timeline_trigger();
