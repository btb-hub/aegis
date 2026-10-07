DROP TRIGGER incident_channel_timeline ON timeline_events;
DROP FUNCTION incident_channel_timeline_trigger();
DROP FUNCTION queue_incident_channel_event(UUID,UUID,TEXT,UUID,TIMESTAMPTZ);
DROP TABLE chat_ack_requests;
DROP TABLE incident_channel_deliveries;
DROP TABLE incident_channel_events;
