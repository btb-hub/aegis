ALTER TABLE teams
    DROP COLUMN IF EXISTS oncall_announced_user_ids,
    DROP COLUMN IF EXISTS slack_channel_id,
    DROP COLUMN IF EXISTS express_chat_id;
