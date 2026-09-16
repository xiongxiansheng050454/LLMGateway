-- name: ListChannelModels :many
SELECT id, model_name, upstream_model, enabled
FROM channel_models
WHERE channel_id = $1
ORDER BY id;

-- name: GetChannelModel :one
SELECT id, model_name, upstream_model, enabled
FROM channel_models
WHERE channel_id = $1 AND model_name = $2;

-- name: CreateChannelModel :one
INSERT INTO channel_models (channel_id, model_name, upstream_model, enabled)
VALUES (sqlc.arg(channel_id), sqlc.arg(model_name), sqlc.arg(upstream_model), sqlc.arg(enabled))
RETURNING id, model_name, upstream_model, enabled;

-- name: UpdateChannelModel :one
UPDATE channel_models
SET upstream_model = sqlc.arg(upstream_model), enabled = sqlc.arg(enabled), updated_at = now()
WHERE channel_id = sqlc.arg(channel_id) AND id = sqlc.arg(id)
RETURNING id, model_name, upstream_model, enabled;

-- name: DeleteChannelModel :execrows
DELETE FROM channel_models WHERE channel_id = $1 AND id = $2;

-- name: ListCatalogModels :many
SELECT
    cm.model_name,
    c.id AS channel_id,
    c.name AS channel_name,
    cm.upstream_model,
    cm.enabled
FROM channel_models cm
JOIN channels c ON c.id = cm.channel_id
WHERE sqlc.arg(enabled_only)::boolean = false OR cm.enabled = true
ORDER BY cm.model_name, c.priority DESC, c.weight DESC, c.id;
