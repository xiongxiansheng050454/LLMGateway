CREATE TABLE channel_health_buckets (
    channel_id BIGINT NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    bucket_start TIMESTAMPTZ NOT NULL,
    requests BIGINT NOT NULL DEFAULT 0,
    errors BIGINT NOT NULL DEFAULT 0,
    timeouts BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (channel_id, bucket_start),
    CHECK (requests >= 0 AND errors >= 0 AND timeouts >= 0)
);

CREATE TABLE channel_breaker_configs (
    channel_id BIGINT PRIMARY KEY REFERENCES channels(id) ON DELETE CASCADE,
    window_seconds INTEGER NOT NULL,
    minimum_samples INTEGER NOT NULL,
    error_rate_percent INTEGER NOT NULL,
    timeout_rate_percent INTEGER NOT NULL,
    cooldown_seconds INTEGER NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (window_seconds > 0 AND minimum_samples > 0 AND error_rate_percent BETWEEN 1 AND 100 AND timeout_rate_percent BETWEEN 1 AND 100 AND cooldown_seconds > 0)
);

CREATE TABLE channel_breaker_probes (
    channel_id BIGINT PRIMARY KEY REFERENCES channels(id) ON DELETE CASCADE,
    lease_id UUID NOT NULL,
    leased_until TIMESTAMPTZ NOT NULL
);
