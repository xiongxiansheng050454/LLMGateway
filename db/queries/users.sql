-- name: ListUsers :many
SELECT
    u.id,
    u.nickname,
    u.user_group,
    u.status,
    b.available_balance,
    b.frozen_balance
FROM users u
LEFT JOIN user_balances b ON b.user_id = u.id
ORDER BY u.id
LIMIT $1 OFFSET $2;

-- name: CountUsers :one
SELECT count(*)::int FROM users;

-- name: ListUserKeys :many
SELECT id, user_id, key_name, prefix, is_active, last_used_at, expires_at
FROM client_api_keys
WHERE user_id = $1
ORDER BY id
LIMIT $2 OFFSET $3;

-- name: ListKeys :many
SELECT id, user_id, key_name, prefix, is_active, last_used_at, expires_at
FROM client_api_keys
ORDER BY id
LIMIT $1 OFFSET $2;

-- name: GetUserBalance :one
SELECT user_id, available_balance, frozen_balance
FROM user_balances
WHERE user_id = $1;

-- name: ListBalanceTransactions :many
SELECT id, user_id, tx_type, amount, balance_after, related_order_id, description, created_at
FROM balance_transactions
WHERE user_id = $1
ORDER BY id DESC
LIMIT $2 OFFSET $3;
