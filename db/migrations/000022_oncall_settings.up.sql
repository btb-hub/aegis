CREATE TABLE oncall_publication_settings (
 id BOOLEAN PRIMARY KEY DEFAULT true CHECK(id),
 time TEXT NOT NULL DEFAULT '03:00',
 timezone TEXT NOT NULL DEFAULT 'UTC',
 next_run_at TIMESTAMPTZ NOT NULL,
 updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
INSERT INTO oncall_publication_settings(next_run_at)
VALUES (date_trunc('day',now() AT TIME ZONE 'UTC') AT TIME ZONE 'UTC' + interval '3 hours' + CASE WHEN (now() AT TIME ZONE 'UTC')::time >= time '03:00' THEN interval '1 day' ELSE interval '0 day' END);
CREATE TABLE oncall_publication_runs (
 local_date DATE PRIMARY KEY,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
