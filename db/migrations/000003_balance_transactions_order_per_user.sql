-- Recharge idempotency is scoped per user: the same related_order_id may be
-- used by different users, but must be unique within a user. Replaces the
-- global unique index created in 000001.
DROP INDEX IF EXISTS balance_transactions_related_order_id_idx;

CREATE UNIQUE INDEX balance_transactions_user_order_idx
    ON balance_transactions (user_id, related_order_id)
    WHERE related_order_id IS NOT NULL AND related_order_id <> '';
