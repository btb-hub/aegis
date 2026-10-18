BEGIN;
CREATE TABLE IF NOT EXISTS sandbox_seed_marks(name text PRIMARY KEY);
INSERT INTO users(id,provider,provider_sub,email,display_name,role,slack_user_id,express_user_huid)
VALUES ('10000000-0000-4000-8000-000000000001','dev','dev-local','dev@localhost','Local Dev User','admin','U_LOCAL_SANDBOX','10000000-0000-4000-8000-000000000002')
ON CONFLICT(provider,provider_sub) DO NOTHING;
INSERT INTO teams(id,workspace_id,name,description,support_tier)
VALUES ('10000000-0000-4000-8000-000000000003','00000000-0000-0000-0000-000000000001','Local sandbox','Local alerts and Mimir anomalies','l3') ON CONFLICT DO NOTHING;
INSERT INTO routing_rules(id,workspace_id,team_id,match_labels,priority)
VALUES ('10000000-0000-4000-8000-000000000004','00000000-0000-0000-0000-000000000001','10000000-0000-4000-8000-000000000003','{"team":"platform"}',100) ON CONFLICT DO NOTHING;
INSERT INTO team_memberships(team_id,user_id)
SELECT '10000000-0000-4000-8000-000000000003',id FROM users WHERE provider='dev' AND provider_sub='dev-local' ON CONFLICT DO NOTHING;
INSERT INTO overrides(id,team_id,user_id,start_at,end_at)
SELECT '10000000-0000-4000-8000-000000000005','10000000-0000-4000-8000-000000000003',id,now()-interval '1 hour',now()+interval '30 days' FROM users WHERE provider='dev' AND provider_sub='dev-local'
ON CONFLICT(id) DO UPDATE SET end_at=EXCLUDED.end_at;
INSERT INTO on_call_slots(id,team_id,user_id,start_at,end_at,source)
SELECT '10000000-0000-4000-8000-000000000006','10000000-0000-4000-8000-000000000003',id,now()-interval '1 hour',now()+interval '30 days','override' FROM users WHERE provider='dev' AND provider_sub='dev-local'
ON CONFLICT(id) DO UPDATE SET end_at=EXCLUDED.end_at;
UPDATE integrations SET mode='custom',enabled=true,
config='{"bot_token":"local-sandbox-slack","signing_secret":"local-sandbox-slack-signing","api_base_url":"http://controls:8081"}'
WHERE workspace_id='00000000-0000-0000-0000-000000000001' AND kind='slack'
AND NOT EXISTS(SELECT 1 FROM sandbox_seed_marks WHERE name='chat_connectors');
UPDATE integrations SET mode='custom',enabled=true,
config='{"bot_id":"10000000-0000-4000-8000-000000000007","secret_key":"local-sandbox-express","host":"http://controls:8081"}'
WHERE workspace_id='00000000-0000-0000-0000-000000000001' AND kind='express'
AND NOT EXISTS(SELECT 1 FROM sandbox_seed_marks WHERE name='chat_connectors');
INSERT INTO sandbox_seed_marks(name) VALUES('chat_connectors') ON CONFLICT DO NOTHING;
COMMIT;
