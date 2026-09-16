-- Deleting a user must also delete their usage logs (see issue #4 decision A).
-- daily_usage_stats.user_id is already ON DELETE CASCADE in 000001.
ALTER TABLE usage_logs
    DROP CONSTRAINT IF EXISTS usage_logs_user_id_fkey,
    ADD CONSTRAINT usage_logs_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
