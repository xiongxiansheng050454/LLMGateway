-- name: ListChannels :many
SELECT
    c.id,
    c.name,
    c.base_url,
    c.auth_type,
    c.status,
    c.weight,
    c.priority,
    c.balance,
    count(cm.id)::int AS model_count
FROM channels c
LEFT JOIN channel_models cm ON cm.channel_id = c.id
GROUP BY c.id
ORDER BY c.id;

-- name: GetChannelSecret :one
SELECT id, name, base_url, api_key_ciphertext, auth_type, status, weight, priority, balance
FROM channels
WHERE id = $1;
