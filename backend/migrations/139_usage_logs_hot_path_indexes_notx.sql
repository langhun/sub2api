-- Add online indexes for high-volume usage_logs read paths.
-- Keep this migration non-transactional because PostgreSQL requires
-- CREATE INDEX CONCURRENTLY to run outside an explicit transaction.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_usage_logs_created_id_desc
ON usage_logs (created_at DESC, id DESC);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_usage_logs_account_created_id_desc
ON usage_logs (account_id, created_at DESC, id DESC)
WHERE account_id IS NOT NULL;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_usage_logs_group_created_id_desc_not_null
ON usage_logs (group_id, created_at DESC, id DESC)
WHERE group_id IS NOT NULL;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_usage_logs_duration_tail_created
ON usage_logs (duration_ms DESC, created_at DESC, id DESC)
WHERE duration_ms IS NOT NULL;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_usage_logs_ttft_tail_created
ON usage_logs (first_token_ms DESC, duration_ms DESC, created_at DESC, id DESC)
WHERE first_token_ms IS NOT NULL;
