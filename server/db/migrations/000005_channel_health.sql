-- Per-channel circuit breaker state. A missing row means "closed".
CREATE TABLE channel_health (
    channel_id BIGINT PRIMARY KEY REFERENCES channels(id) ON DELETE CASCADE,
    state TEXT NOT NULL DEFAULT 'closed' CHECK (state IN ('closed', 'open', 'half-open')),
    consecutive_failures INTEGER NOT NULL DEFAULT 0,
    success_count BIGINT NOT NULL DEFAULT 0,
    failure_count BIGINT NOT NULL DEFAULT 0,
    opened_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
