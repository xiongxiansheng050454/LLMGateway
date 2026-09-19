-- Queueing synchronous proxy requests is unsupported. Preserve existing rules
-- by converting them to reject before constraining new writes.
UPDATE rate_limit_rules
SET action = 'reject', extras = extras - 'queue_timeout_seconds'
WHERE action = 'queue';

ALTER TABLE rate_limit_rules
    DROP CONSTRAINT IF EXISTS rate_limit_rules_action_check,
    ADD CONSTRAINT rate_limit_rules_action_check CHECK (action = 'reject');
