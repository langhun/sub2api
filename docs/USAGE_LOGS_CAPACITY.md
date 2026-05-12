# usage_logs Capacity and Upgrade Notes

This guide is for high-traffic deployments where `usage_logs` becomes one of the largest PostgreSQL tables. It covers query-plan checks, the online indexes added for hot paths, retention/partitioning choices, and URL security defaults that can affect upgrades.

## Online indexes

Migration `backend/migrations/139_usage_logs_hot_path_indexes_notx.sql` adds only concurrent, idempotent indexes:

| Index | Main path |
|-------|-----------|
| `idx_usage_logs_created_id_desc` | Time-window and recent-log reads ordered by newest rows |
| `idx_usage_logs_account_created_id_desc` | Account-scoped log pages ordered by newest IDs |
| `idx_usage_logs_group_created_id_desc_not_null` | Group-scoped log pages ordered by newest IDs |
| `idx_usage_logs_duration_tail_created` | Slow-tail drilldowns ordered by `duration_ms` |
| `idx_usage_logs_ttft_tail_created` | TTFT drilldowns ordered by `first_token_ms` |

These do not replace existing aggregation indexes such as `(account_id, created_at)`, `(api_key_id, created_at)`, `(model, created_at)`, and the partial `(group_id, created_at)` index. Keep those existing indexes because they serve range aggregations and billing/statistics queries.

The migration uses `CREATE INDEX CONCURRENTLY IF NOT EXISTS`, so normal reads and writes can continue while PostgreSQL builds the indexes. Run it during a quieter window if the table is already very large, because concurrent builds still consume I/O and CPU.

## Verify with EXPLAIN

Run `EXPLAIN (ANALYZE, BUFFERS)` on the same PostgreSQL instance and with realistic time ranges. A good plan should avoid a large sequential scan over the whole table and should keep shared buffer reads proportional to the requested window.

Recent time-window page:

```sql
EXPLAIN (ANALYZE, BUFFERS)
SELECT id, created_at, user_id, api_key_id, account_id, group_id, duration_ms
FROM usage_logs
WHERE created_at >= now() - interval '24 hours'
ORDER BY created_at DESC, id DESC
LIMIT 100;
```

Account-scoped newest rows:

```sql
EXPLAIN (ANALYZE, BUFFERS)
SELECT id, created_at, account_id, model, actual_cost
FROM usage_logs
WHERE account_id = 123
ORDER BY id DESC
LIMIT 100;
```

Group-scoped newest rows:

```sql
EXPLAIN (ANALYZE, BUFFERS)
SELECT id, created_at, group_id, model, actual_cost
FROM usage_logs
WHERE group_id = 456
ORDER BY id DESC
LIMIT 100;
```

Slow-tail drilldown:

```sql
EXPLAIN (ANALYZE, BUFFERS)
SELECT id, created_at, account_id, group_id, duration_ms
FROM usage_logs
WHERE created_at >= now() - interval '24 hours'
  AND duration_ms IS NOT NULL
ORDER BY duration_ms DESC NULLS LAST, created_at DESC, id DESC
LIMIT 100;
```

TTFT drilldown:

```sql
EXPLAIN (ANALYZE, BUFFERS)
SELECT id, created_at, account_id, group_id, first_token_ms
FROM usage_logs
WHERE created_at >= now() - interval '24 hours'
  AND first_token_ms IS NOT NULL
ORDER BY first_token_ms DESC NULLS LAST, duration_ms DESC NULLS LAST, created_at DESC
LIMIT 100;
```

Index inventory check:

```sql
SELECT indexname, indexdef
FROM pg_indexes
WHERE schemaname = 'public'
  AND tablename = 'usage_logs'
ORDER BY indexname;
```

Table size check:

```sql
SELECT
  pg_size_pretty(pg_total_relation_size('usage_logs')) AS total_size,
  pg_size_pretty(pg_relation_size('usage_logs')) AS heap_size,
  pg_size_pretty(pg_indexes_size('usage_logs')) AS index_size;
```

## Retention policy

For high-traffic installations, avoid keeping unlimited raw `usage_logs` rows.

- Keep raw rows only for the period that needs drilldown and audit detail, commonly 30-90 days.
- Preserve long-term summaries in existing rollup/analytics tables before deleting raw rows.
- Delete in small batches by `created_at` or `id` to avoid long locks and huge WAL spikes.
- Run `VACUUM (ANALYZE) usage_logs` after large deletes; use `VACUUM FULL` only in a planned maintenance window because it rewrites the table.
- Keep a database backup before changing retention for the first time.

Example batch delete pattern:

```sql
WITH doomed AS (
  SELECT id
  FROM usage_logs
  WHERE created_at < now() - interval '90 days'
  ORDER BY id
  LIMIT 5000
)
DELETE FROM usage_logs
WHERE id IN (SELECT id FROM doomed);
```

Repeat the batch until it deletes zero rows, then analyze:

```sql
VACUUM (ANALYZE) usage_logs;
```

## Partitioning guidance

Consider range partitioning by `created_at` when either condition is true:

- raw `usage_logs` grows beyond the size where retention deletes take too long for your maintenance window;
- daily write volume makes recent dashboard/detail queries compete with old historical rows.

Recommended shape:

- partition by month for moderate traffic, by day for very high traffic;
- create matching local indexes on each partition for the hot paths above;
- drop old partitions instead of deleting rows when the retention window expires;
- rehearse conversion on a clone first, because converting an existing large table requires a planned migration and application downtime or a dual-write/backfill strategy.

Do not partition only because the table is large. Partition when it shortens retention operations or measurably improves plans for your real queries.

## URL security defaults during upgrades

Current secure defaults keep hostname allowlist checks enabled and reject insecure or private targets unless explicitly opted in. When `security.url_allowlist.enabled=false`, the hostname allowlist is disabled, but HTTP URLs and private/loopback hosts are still blocked unless both relevant flags are set.

If an older deployment intentionally uses local or internal HTTP upstreams, set the compatibility flags before upgrading:

```yaml
security:
  url_allowlist:
    enabled: false
    allow_insecure_http: true
    allow_private_hosts: true
```

Equivalent environment variables:

```bash
SECURITY_URL_ALLOWLIST_ENABLED=false
SECURITY_URL_ALLOWLIST_ALLOW_INSECURE_HTTP=true
SECURITY_URL_ALLOWLIST_ALLOW_PRIVATE_HOSTS=true
```

Use this only for trusted internal networks. For production internet-facing deployments, prefer HTTPS upstreams and a narrow hostname allowlist.
