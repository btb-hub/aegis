CREATE TABLE application_settings (
    singleton BOOLEAN PRIMARY KEY DEFAULT true CHECK (singleton),
    values JSONB NOT NULL DEFAULT '{}',
    enabled JSONB NOT NULL DEFAULT '{}',
    revision BIGINT NOT NULL DEFAULT 1,
    imported BOOLEAN NOT NULL DEFAULT false,
    bootstrap_closed BOOLEAN NOT NULL DEFAULT false,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE settings_provider_drafts (
    provider TEXT PRIMARY KEY CHECK (provider IN ('google','slack','express')),
    values JSONB NOT NULL,
    revision BIGINT NOT NULL DEFAULT 1,
    tested_revision BIGINT,
    tested_by TEXT,
    tested_at TIMESTAMPTZ,
    expected_email TEXT NOT NULL DEFAULT ''
);
CREATE TABLE settings_bootstrap_sessions (
    token_hash TEXT PRIMARY KEY,
    expires_at TIMESTAMPTZ NOT NULL
);
CREATE TABLE settings_authorizations (
    state_hash TEXT PRIMARY KEY,
    kind TEXT NOT NULL CHECK (kind IN ('login','provider_test','bootstrap')),
    provider TEXT NOT NULL CHECK (provider IN ('google','slack','express')),
    nonce TEXT NOT NULL,
    session_hash TEXT NOT NULL,
    expected_email TEXT NOT NULL DEFAULT '',
    draft_revision BIGINT NOT NULL DEFAULT 0,
    settings_revision BIGINT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    exchanging BOOLEAN NOT NULL DEFAULT false
);
CREATE INDEX settings_authorizations_expiry_idx ON settings_authorizations(expires_at);

-- Existing connector APIs retain their response shape. Audit every credential/config change
-- by field name only, including legacy writers, without copying values into the audit log.
CREATE FUNCTION audit_integration_settings() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE row_id uuid; fields text[];
BEGIN
    IF TG_OP='UPDATE' THEN
        fields:=ARRAY[]::text[];
        IF OLD.config IS DISTINCT FROM NEW.config THEN fields:=array_append(fields,'config'); END IF;
        IF OLD.enabled IS DISTINCT FROM NEW.enabled THEN fields:=array_append(fields,'enabled'); END IF;
        IF OLD.name IS DISTINCT FROM NEW.name THEN fields:=array_append(fields,'name'); END IF;
        IF OLD.mode IS DISTINCT FROM NEW.mode THEN fields:=array_append(fields,'mode'); END IF;
        IF cardinality(fields)=0 THEN RETURN NEW; END IF;
    ELSE
        fields:=ARRAY['config','enabled','name','mode'];
    END IF;
    IF TG_OP='DELETE' THEN row_id:=OLD.id; ELSE row_id:=NEW.id; END IF;
    INSERT INTO audit_log(action,resource_type,resource_id,details)
    VALUES('settings.integration_changed','integration',row_id,jsonb_build_object('operation',TG_OP,'fields',fields));
    RETURN COALESCE(NEW,OLD);
END;
$$;
CREATE TRIGGER integration_settings_audit AFTER INSERT OR UPDATE OR DELETE ON integrations
FOR EACH ROW EXECUTE FUNCTION audit_integration_settings();
