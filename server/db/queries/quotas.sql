-- name: GetQuotaPolicy :one
SELECT id, policy_name, scope_type, user_id, api_key_id, period_type,
       token_limit, cost_limit::text AS cost_limit, enabled
FROM quota_policies
WHERE id = $1 AND deleted_at IS NULL;

-- name: CreateQuotaPolicy :one
INSERT INTO quota_policies (policy_name, scope_type, user_id, api_key_id, period_type, token_limit, cost_limit, enabled)
VALUES (sqlc.arg(policy_name), sqlc.arg(scope_type), sqlc.narg(user_id), sqlc.narg(api_key_id),
        sqlc.arg(period_type), sqlc.narg(token_limit), sqlc.narg(cost_limit), sqlc.arg(enabled))
RETURNING id;

-- name: UpdateQuotaPolicy :execrows
UPDATE quota_policies
SET policy_name = sqlc.arg(policy_name), token_limit = sqlc.narg(token_limit),
    cost_limit = sqlc.narg(cost_limit), enabled = sqlc.arg(enabled), updated_at = now()
WHERE id = sqlc.arg(id) AND deleted_at IS NULL;

-- name: DeleteQuotaPolicy :execrows
UPDATE quota_policies
SET enabled = false, deleted_at = now(), updated_at = now()
WHERE id = $1 AND deleted_at IS NULL;
