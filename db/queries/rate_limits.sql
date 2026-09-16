-- name: ListRateLimitRules :many
SELECT id, rule_name, target_type, target_value, metric, limit_value, window_seconds, action, priority, enabled, extras
FROM rate_limit_rules
WHERE sqlc.narg(enabled)::boolean IS NULL OR enabled = sqlc.narg(enabled)::boolean
ORDER BY priority, id
LIMIT $1 OFFSET $2;

-- name: CountRateLimitRules :one
SELECT count(*)::int
FROM rate_limit_rules
WHERE sqlc.narg(enabled)::boolean IS NULL OR enabled = sqlc.narg(enabled)::boolean;
