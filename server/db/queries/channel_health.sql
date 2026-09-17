-- name: GetChannelHealth :one
SELECT channel_id, state, consecutive_failures, success_count, failure_count, opened_at, updated_at
FROM channel_health
WHERE channel_id = $1;

-- name: EnsureChannelHealth :exec
INSERT INTO channel_health (channel_id)
VALUES ($1)
ON CONFLICT (channel_id) DO NOTHING;

-- name: GetChannelHealthForUpdate :one
SELECT channel_id, state, consecutive_failures, success_count, failure_count, opened_at, updated_at
FROM channel_health
WHERE channel_id = $1
FOR UPDATE;

-- name: UpdateChannelHealth :execrows
UPDATE channel_health
SET state = sqlc.arg(state),
    consecutive_failures = sqlc.arg(consecutive_failures),
    success_count = sqlc.arg(success_count),
    failure_count = sqlc.arg(failure_count),
    opened_at = sqlc.narg(opened_at),
    updated_at = now()
WHERE channel_id = sqlc.arg(channel_id);

-- name: DeleteChannelHealth :execrows
DELETE FROM channel_health WHERE channel_id = $1;

-- name: ListChannelHealth :many
SELECT channel_id, state, consecutive_failures, success_count, failure_count, opened_at, updated_at
FROM channel_health
ORDER BY channel_id;
