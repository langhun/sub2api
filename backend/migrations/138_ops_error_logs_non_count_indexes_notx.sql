-- Support monitoring error-only paths that read non-count-token ops errors.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_ops_error_logs_non_count_created_model
ON ops_error_logs (created_at, (COALESCE(requested_model, model)))
WHERE is_count_tokens = false
  AND COALESCE(requested_model, model) IS NOT NULL;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_ops_error_logs_non_count_created_group_model
ON ops_error_logs (created_at, group_id, (COALESCE(requested_model, model)))
WHERE is_count_tokens = false
  AND group_id IS NOT NULL
  AND COALESCE(requested_model, model) IS NOT NULL;
