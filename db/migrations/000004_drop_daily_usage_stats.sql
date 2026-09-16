-- daily_usage_stats is unused: statistics are aggregated from usage_logs in
-- real time. Its composite primary key (stat_date, user_id, channel_id, model)
-- also implicitly forced user_id/channel_id to NOT NULL, so it could not store
-- global per-day rows. Drop it instead of keeping a misleading dead table.
DROP TABLE IF EXISTS daily_usage_stats;
