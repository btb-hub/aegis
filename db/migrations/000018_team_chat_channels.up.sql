ALTER TABLE teams
    ADD COLUMN express_chat_id TEXT,
    ADD COLUMN slack_channel_id TEXT,
    ADD COLUMN oncall_announced_user_ids TEXT;
