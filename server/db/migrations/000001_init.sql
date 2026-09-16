CREATE TABLE channels (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    base_url TEXT NOT NULL,
    api_key_ciphertext TEXT NOT NULL,
    auth_type TEXT NOT NULL DEFAULT 'bearer',
    status INTEGER NOT NULL DEFAULT 1,
    weight INTEGER NOT NULL DEFAULT 100,
    priority INTEGER NOT NULL DEFAULT 0,
    balance NUMERIC(20, 6),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE channel_models (
    id BIGSERIAL PRIMARY KEY,
    channel_id BIGINT NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    model_name TEXT NOT NULL,
    upstream_model TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (channel_id, model_name)
);

CREATE TABLE model_pricing (
    id BIGSERIAL PRIMARY KEY,
    channel_id BIGINT NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    model_name TEXT NOT NULL,
    input_price_per_1m NUMERIC(20, 8) NOT NULL,
    output_price_per_1m NUMERIC(20, 8) NOT NULL,
    cached_input_price_per_1m NUMERIC(20, 8),
    currency TEXT NOT NULL DEFAULT 'USD',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (channel_id, model_name)
);

CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    nickname TEXT NOT NULL DEFAULT '',
    user_group TEXT NOT NULL DEFAULT 'default',
    status TEXT NOT NULL DEFAULT 'active',
    password_hash TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (status IN ('active', 'suspended'))
);

CREATE TABLE user_balances (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    available_balance NUMERIC(20, 6) NOT NULL DEFAULT 0,
    frozen_balance NUMERIC(20, 6) NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE balance_transactions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tx_type TEXT NOT NULL,
    amount NUMERIC(20, 6) NOT NULL,
    balance_after NUMERIC(20, 6) NOT NULL,
    related_order_id TEXT,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX balance_transactions_related_order_id_idx
    ON balance_transactions (related_order_id)
    WHERE related_order_id IS NOT NULL AND related_order_id <> '';

CREATE TABLE client_api_keys (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key_name TEXT NOT NULL DEFAULT 'default',
    prefix TEXT NOT NULL DEFAULT 'sk-',
    key_hash TEXT NOT NULL UNIQUE,
    permissions JSONB NOT NULL DEFAULT '{"models":["*"]}'::jsonb,
    rate_limit_overrides JSONB,
    expires_at TIMESTAMPTZ,
    is_active BOOLEAN NOT NULL DEFAULT true,
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE rate_limit_rules (
    id BIGSERIAL PRIMARY KEY,
    rule_name TEXT NOT NULL,
    target_type TEXT NOT NULL,
    target_value TEXT NOT NULL DEFAULT '*',
    metric TEXT NOT NULL,
    limit_value BIGINT NOT NULL,
    window_seconds INTEGER NOT NULL,
    action TEXT NOT NULL DEFAULT 'reject',
    priority INTEGER NOT NULL DEFAULT 100,
    enabled BOOLEAN NOT NULL DEFAULT true,
    extras JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (target_type IN ('global', 'user', 'api_key', 'model', 'channel')),
    CHECK (metric IN ('rpm', 'tpm', 'rpd', 'tpd', 'concurrency')),
    CHECK (action IN ('reject', 'queue'))
);

CREATE INDEX rate_limit_rules_enabled_idx ON rate_limit_rules (enabled, priority, id);

CREATE TABLE usage_logs (
    id BIGSERIAL PRIMARY KEY,
    request_id TEXT NOT NULL UNIQUE,
    user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    api_key_id BIGINT REFERENCES client_api_keys(id) ON DELETE SET NULL,
    channel_id BIGINT REFERENCES channels(id) ON DELETE SET NULL,
    model TEXT NOT NULL,
    upstream_model TEXT NOT NULL DEFAULT '',
    input_tokens BIGINT NOT NULL DEFAULT 0,
    output_tokens BIGINT NOT NULL DEFAULT 0,
    cached_input_tokens BIGINT NOT NULL DEFAULT 0,
    total_tokens BIGINT NOT NULL DEFAULT 0,
    unit_price_input_per_1m NUMERIC(20, 8) NOT NULL DEFAULT 0,
    unit_price_output_per_1m NUMERIC(20, 8) NOT NULL DEFAULT 0,
    total_cost NUMERIC(20, 6) NOT NULL DEFAULT 0,
    duration_ms BIGINT NOT NULL DEFAULT 0,
    ttft_ms BIGINT,
    status TEXT NOT NULL,
    error_code TEXT NOT NULL DEFAULT '',
    client_ip TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX usage_logs_created_at_idx ON usage_logs (created_at DESC, id DESC);
CREATE INDEX usage_logs_user_created_at_idx ON usage_logs (user_id, created_at DESC);
CREATE INDEX usage_logs_channel_created_at_idx ON usage_logs (channel_id, created_at DESC);
CREATE INDEX usage_logs_model_created_at_idx ON usage_logs (model, created_at DESC);

CREATE TABLE daily_usage_stats (
    stat_date DATE NOT NULL,
    user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
    channel_id BIGINT REFERENCES channels(id) ON DELETE CASCADE,
    model TEXT NOT NULL DEFAULT '',
    request_count BIGINT NOT NULL DEFAULT 0,
    success_count BIGINT NOT NULL DEFAULT 0,
    error_count BIGINT NOT NULL DEFAULT 0,
    total_tokens BIGINT NOT NULL DEFAULT 0,
    total_cost NUMERIC(20, 6) NOT NULL DEFAULT 0,
    PRIMARY KEY (stat_date, user_id, channel_id, model)
);
