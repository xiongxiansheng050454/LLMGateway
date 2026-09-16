-- name: ListUsageLogs :many
SELECT
    l.id,
    l.request_id,
    l.user_id,
    l.api_key_id,
    l.channel_id,
    c.name AS channel_name,
    l.model,
    l.upstream_model,
    l.input_tokens,
    l.output_tokens,
    l.cached_input_tokens,
    l.total_tokens,
    l.unit_price_input_per_1m,
    l.unit_price_output_per_1m,
    l.total_cost,
    l.duration_ms,
    l.ttft_ms,
    l.status,
    l.error_code,
    l.client_ip,
    l.created_at
FROM usage_logs l
LEFT JOIN channels c ON c.id = l.channel_id
ORDER BY l.created_at DESC, l.id DESC
LIMIT $1 OFFSET $2;

-- name: GetUsageLog :one
SELECT
    l.id,
    l.request_id,
    l.user_id,
    l.api_key_id,
    l.channel_id,
    c.name AS channel_name,
    l.model,
    l.upstream_model,
    l.input_tokens,
    l.output_tokens,
    l.cached_input_tokens,
    l.total_tokens,
    l.unit_price_input_per_1m,
    l.unit_price_output_per_1m,
    l.total_cost,
    l.duration_ms,
    l.ttft_ms,
    l.status,
    l.error_code,
    l.client_ip,
    l.created_at
FROM usage_logs l
LEFT JOIN channels c ON c.id = l.channel_id
WHERE l.id = $1;

-- name: StatsOverview :one
SELECT
    count(*)::bigint AS request_count,
    count(*) FILTER (WHERE status = 'success')::bigint AS success_count,
    count(*) FILTER (WHERE status <> 'success')::bigint AS error_count,
    coalesce(sum(total_tokens), 0)::bigint AS total_tokens,
    coalesce(sum(total_cost), 0)::numeric(20, 6) AS total_cost,
    count(DISTINCT user_id)::bigint AS active_user_count
FROM usage_logs
WHERE created_at >= $1 AND created_at <= $2;

-- name: StatsChannels :many
SELECT
    l.channel_id,
    coalesce(c.name, '') AS channel_name,
    count(*)::bigint AS request_count,
    count(*) FILTER (WHERE l.status = 'success')::bigint AS success_count,
    count(*) FILTER (WHERE l.status <> 'success')::bigint AS error_count,
    coalesce(sum(l.total_tokens), 0)::bigint AS total_tokens,
    coalesce(sum(l.total_cost), 0)::numeric(20, 6) AS total_cost
FROM usage_logs l
LEFT JOIN channels c ON c.id = l.channel_id
WHERE l.created_at >= $1 AND l.created_at <= $2
GROUP BY l.channel_id, c.name
ORDER BY request_count DESC, l.channel_id;
