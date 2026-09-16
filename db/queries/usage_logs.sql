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
    l.unit_price_input_per_1m::text AS unit_price_input_per_1m,
    l.unit_price_output_per_1m::text AS unit_price_output_per_1m,
    l.total_cost::text AS total_cost,
    l.duration_ms,
    l.ttft_ms,
    l.status,
    l.error_code,
    l.client_ip,
    l.created_at
FROM usage_logs l
LEFT JOIN channels c ON c.id = l.channel_id
WHERE (sqlc.narg(user_id)::bigint IS NULL OR l.user_id = sqlc.narg(user_id)::bigint)
  AND (sqlc.narg(channel_id)::bigint IS NULL OR l.channel_id = sqlc.narg(channel_id)::bigint)
  AND (sqlc.narg(model)::text IS NULL OR l.model = sqlc.narg(model)::text)
  AND (sqlc.narg(status)::text IS NULL OR l.status = sqlc.narg(status)::text)
  AND (sqlc.narg(start_time)::timestamptz IS NULL OR l.created_at >= sqlc.narg(start_time)::timestamptz)
  AND (sqlc.narg(end_time)::timestamptz IS NULL OR l.created_at <= sqlc.narg(end_time)::timestamptz)
ORDER BY l.created_at DESC, l.id DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountUsageLogs :one
SELECT count(*)::int
FROM usage_logs l
WHERE (sqlc.narg(user_id)::bigint IS NULL OR l.user_id = sqlc.narg(user_id)::bigint)
  AND (sqlc.narg(channel_id)::bigint IS NULL OR l.channel_id = sqlc.narg(channel_id)::bigint)
  AND (sqlc.narg(model)::text IS NULL OR l.model = sqlc.narg(model)::text)
  AND (sqlc.narg(status)::text IS NULL OR l.status = sqlc.narg(status)::text)
  AND (sqlc.narg(start_time)::timestamptz IS NULL OR l.created_at >= sqlc.narg(start_time)::timestamptz)
  AND (sqlc.narg(end_time)::timestamptz IS NULL OR l.created_at <= sqlc.narg(end_time)::timestamptz);

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
    l.unit_price_input_per_1m::text AS unit_price_input_per_1m,
    l.unit_price_output_per_1m::text AS unit_price_output_per_1m,
    l.total_cost::text AS total_cost,
    l.duration_ms,
    l.ttft_ms,
    l.status,
    l.error_code,
    l.client_ip,
    l.created_at
FROM usage_logs l
LEFT JOIN channels c ON c.id = l.channel_id
WHERE l.id = $1;

-- name: InsertUsageLog :one
INSERT INTO usage_logs (
    request_id, user_id, api_key_id, channel_id, model, upstream_model,
    input_tokens, output_tokens, cached_input_tokens, total_tokens,
    unit_price_input_per_1m, unit_price_output_per_1m, total_cost,
    duration_ms, ttft_ms, status, error_code, client_ip
)
VALUES (
    sqlc.arg(request_id),
    sqlc.narg(user_id),
    sqlc.narg(api_key_id),
    sqlc.narg(channel_id),
    sqlc.arg(model),
    sqlc.arg(upstream_model),
    sqlc.arg(input_tokens),
    sqlc.arg(output_tokens),
    sqlc.arg(cached_input_tokens),
    sqlc.arg(total_tokens),
    COALESCE(NULLIF(sqlc.arg(unit_price_input_per_1m), '')::numeric, 0),
    COALESCE(NULLIF(sqlc.arg(unit_price_output_per_1m), '')::numeric, 0),
    COALESCE(NULLIF(sqlc.arg(total_cost), '')::numeric, 0),
    sqlc.arg(duration_ms),
    sqlc.narg(ttft_ms),
    sqlc.arg(status),
    sqlc.arg(error_code),
    sqlc.arg(client_ip)
)
RETURNING id;

-- name: StatsOverview :one
SELECT
    count(*)::bigint AS request_count,
    count(*) FILTER (WHERE status = 'success')::bigint AS success_count,
    count(*) FILTER (WHERE status <> 'success')::bigint AS error_count,
    coalesce(sum(total_tokens), 0)::bigint AS total_tokens,
    coalesce(sum(total_cost), 0)::numeric(20, 6)::text AS total_cost,
    count(DISTINCT user_id)::bigint AS active_user_count
FROM usage_logs
WHERE created_at >= $1 AND created_at <= $2;

-- name: StatsDaily :many
SELECT
    (l.created_at AT TIME ZONE 'UTC')::date AS stat_date,
    count(*)::bigint AS request_count,
    count(*) FILTER (WHERE l.status = 'success')::bigint AS success_count,
    count(*) FILTER (WHERE l.status <> 'success')::bigint AS error_count,
    coalesce(sum(l.total_tokens), 0)::bigint AS total_tokens,
    coalesce(sum(l.total_cost), 0)::numeric(20, 6)::text AS total_cost
FROM usage_logs l
WHERE (l.created_at AT TIME ZONE 'UTC')::date >= sqlc.arg(date_from)::text::date
  AND (l.created_at AT TIME ZONE 'UTC')::date <= sqlc.arg(date_to)::text::date
GROUP BY stat_date
ORDER BY stat_date
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountStatsDaily :one
SELECT count(*)::int FROM (
    SELECT (l.created_at AT TIME ZONE 'UTC')::date AS stat_date
    FROM usage_logs l
    WHERE (l.created_at AT TIME ZONE 'UTC')::date >= sqlc.arg(date_from)::text::date
      AND (l.created_at AT TIME ZONE 'UTC')::date <= sqlc.arg(date_to)::text::date
    GROUP BY stat_date
) AS days;

-- name: StatsChannels :many
SELECT
    l.channel_id,
    coalesce(c.name, '') AS channel_name,
    count(*)::bigint AS request_count,
    count(*) FILTER (WHERE l.status = 'success')::bigint AS success_count,
    count(*) FILTER (WHERE l.status <> 'success')::bigint AS error_count,
    coalesce(sum(l.total_tokens), 0)::bigint AS total_tokens,
    coalesce(sum(l.total_cost), 0)::numeric(20, 6)::text AS total_cost
FROM usage_logs l
LEFT JOIN channels c ON c.id = l.channel_id
WHERE l.created_at >= $1 AND l.created_at <= $2
GROUP BY l.channel_id, c.name
ORDER BY request_count DESC, l.channel_id;
