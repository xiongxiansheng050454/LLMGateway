-- name: ListChannelModels :many
SELECT id, model_name, upstream_model, enabled
FROM channel_models
WHERE channel_id = $1
ORDER BY id;

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
