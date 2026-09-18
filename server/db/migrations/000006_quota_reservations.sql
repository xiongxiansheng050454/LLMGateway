CREATE TABLE quota_policies (
    id BIGSERIAL PRIMARY KEY,
    policy_name TEXT NOT NULL,
    scope_type TEXT NOT NULL,
    user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
    api_key_id BIGINT REFERENCES client_api_keys(id) ON DELETE CASCADE,
    period_type TEXT NOT NULL,
    token_limit BIGINT,
    cost_limit NUMERIC(20, 6),
    enabled BOOLEAN NOT NULL DEFAULT true,
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (scope_type IN ('user', 'api_key')),
    CHECK (period_type IN ('day', 'month')),
    CHECK ((scope_type = 'user' AND user_id IS NOT NULL AND api_key_id IS NULL) OR
           (scope_type = 'api_key' AND user_id IS NULL AND api_key_id IS NOT NULL)),
    CHECK (token_limit IS NULL OR token_limit > 0),
    CHECK (cost_limit IS NULL OR cost_limit > 0),
    CHECK (token_limit IS NOT NULL OR cost_limit IS NOT NULL)
);

ALTER TABLE client_api_keys ADD CONSTRAINT client_api_keys_id_user_id_unique UNIQUE (id, user_id);

CREATE UNIQUE INDEX quota_policies_user_period_uidx
    ON quota_policies (user_id, period_type)
    WHERE scope_type = 'user' AND deleted_at IS NULL;
CREATE UNIQUE INDEX quota_policies_key_period_uidx
    ON quota_policies (api_key_id, period_type)
    WHERE scope_type = 'api_key' AND deleted_at IS NULL;

CREATE TABLE quota_buckets (
    policy_id BIGINT NOT NULL REFERENCES quota_policies(id) ON DELETE CASCADE,
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    used_tokens BIGINT NOT NULL DEFAULT 0,
    reserved_tokens BIGINT NOT NULL DEFAULT 0,
    used_cost NUMERIC(20, 6) NOT NULL DEFAULT 0,
    reserved_cost NUMERIC(20, 6) NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (policy_id, period_start),
    CHECK (period_end > period_start),
    CHECK (used_tokens >= 0 AND reserved_tokens >= 0),
    CHECK (used_cost >= 0 AND reserved_cost >= 0)
);

CREATE TABLE quota_reservations (
    id BIGSERIAL PRIMARY KEY,
    request_id TEXT NOT NULL UNIQUE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    api_key_id BIGINT NOT NULL,
    model TEXT NOT NULL DEFAULT '',
    estimated_tokens BIGINT NOT NULL,
    estimated_cost NUMERIC(20, 6) NOT NULL,
    actual_tokens BIGINT,
    actual_cost NUMERIC(20, 6),
    status TEXT NOT NULL DEFAULT 'pending',
    expires_at TIMESTAMPTZ NOT NULL,
    settled_at TIMESTAMPTZ,
    released_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (estimated_tokens >= 0 AND estimated_cost >= 0),
    CHECK (actual_tokens IS NULL OR actual_tokens >= 0),
    CHECK (actual_cost IS NULL OR actual_cost >= 0),
    CHECK (status IN ('pending', 'settled', 'released', 'expired'))
);

ALTER TABLE quota_reservations
    ADD CONSTRAINT quota_reservations_key_user_fkey
    FOREIGN KEY (api_key_id, user_id) REFERENCES client_api_keys(id, user_id) ON DELETE RESTRICT;

CREATE INDEX quota_reservations_pending_expiry_idx
    ON quota_reservations (expires_at, id) WHERE status = 'pending';
CREATE INDEX quota_reservations_scope_idx
    ON quota_reservations (user_id, api_key_id, status, expires_at);

CREATE TABLE quota_reservation_items (
    reservation_id BIGINT NOT NULL REFERENCES quota_reservations(id) ON DELETE CASCADE,
    policy_id BIGINT NOT NULL,
    period_start TIMESTAMPTZ NOT NULL,
    reserved_tokens BIGINT NOT NULL,
    reserved_cost NUMERIC(20, 6) NOT NULL,
    actual_tokens BIGINT,
    actual_cost NUMERIC(20, 6),
    PRIMARY KEY (reservation_id, policy_id),
    FOREIGN KEY (policy_id, period_start) REFERENCES quota_buckets(policy_id, period_start) ON DELETE CASCADE,
    CHECK (reserved_tokens >= 0 AND reserved_cost >= 0)
);
