-- name: GetChannelHealth :one
SELECT channel_id, state, consecutive_failures, success_count, failure_count, opened_at, updated_at
FROM channel_health
WHERE channel_id = $1;

-- name: UpsertChannelHealth :exec
INSERT INTO channel_health (channel_id, state, consecutive_failures, success_count, failure_count, opened_at, updated_at)
VALUES (
    sqlc.arg(channel_id),
    sqlc.arg(state),
    sqlc.arg(consecutive_failures),
    sqlc.arg(success_count),
    sqlc.arg(failure_count),
    sqlc.narg(opened_at),
    now()
)
ON CONFLICT (channel_id) DO UPDATE SET
    state = EXCLUDED.state,
    consecutive_failures = EXCLUDED.consecutive_failures,
    success_count = EXCLUDED.success_count,
    failure_count = EXCLUDED.failure_count,
    opened_at = EXCLUDED.opened_at,
    updated_at = now();

-- name: DeleteChannelHealth :execrows
DELETE FROM channel_health WHERE channel_id = $1;

-- name: ListChannelHealth :many
SELECT channel_id, state, consecutive_failures, success_count, failure_count, opened_at, updated_at
FROM channel_health
ORDER BY channel_id;
