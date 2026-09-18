CREATE TABLE rate_limit_counters (
    rule_id BIGINT NOT NULL REFERENCES rate_limit_rules(id) ON DELETE CASCADE,
    bucket_start TIMESTAMPTZ NOT NULL,
    current_count BIGINT NOT NULL DEFAULT 0,
    previous_count BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (rule_id, bucket_start),
    CHECK (current_count >= 0 AND previous_count >= 0)
);

CREATE TABLE rate_limit_reservations (
    id BIGSERIAL PRIMARY KEY,
    request_id TEXT NOT NULL UNIQUE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    api_key_id BIGINT NOT NULL REFERENCES client_api_keys(id) ON DELETE CASCADE,
    model TEXT NOT NULL DEFAULT '',
    channel_id BIGINT REFERENCES channels(id) ON DELETE CASCADE,
    estimated_tokens BIGINT NOT NULL DEFAULT 0,
    bucket_starts JSONB NOT NULL DEFAULT '{}'::jsonb,
    status TEXT NOT NULL DEFAULT 'pending',
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    released_at TIMESTAMPTZ,
    CHECK (estimated_tokens >= 0),
    CHECK (status IN ('pending', 'released', 'settled', 'expired'))
);

CREATE INDEX rate_limit_reservations_pending_expiry_idx ON rate_limit_reservations (expires_at, id) WHERE status = 'pending';
