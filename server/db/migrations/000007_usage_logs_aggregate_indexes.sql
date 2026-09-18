-- These indexes match the API Key, model, and channel aggregation windows.
CREATE INDEX usage_logs_api_key_created_at_idx ON usage_logs (api_key_id, created_at DESC);
CREATE INDEX usage_logs_api_key_model_created_at_idx ON usage_logs (api_key_id, model, created_at DESC);
CREATE INDEX usage_logs_api_key_channel_created_at_idx ON usage_logs (api_key_id, channel_id, created_at DESC);
