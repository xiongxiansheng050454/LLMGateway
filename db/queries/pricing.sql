-- name: ListPricing :many
SELECT
    p.id,
    p.channel_id,
    c.name AS channel_name,
    p.model_name,
    cm.upstream_model,
    p.input_price_per_1m,
    p.output_price_per_1m,
    p.cached_input_price_per_1m,
    p.currency
FROM model_pricing p
JOIN channels c ON c.id = p.channel_id
LEFT JOIN channel_models cm ON cm.channel_id = p.channel_id AND cm.model_name = p.model_name
ORDER BY p.id;
