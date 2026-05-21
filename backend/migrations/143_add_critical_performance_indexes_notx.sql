-- Migration: 143_add_critical_performance_indexes_notx
-- Purpose: 添加 5 个关键性能索引，优化高频查询路径
-- Priority: HIGH - 解决数据库审查发现的性能瓶颈
-- Reference: DATABASE_REVIEW.md

-- 1. usage_logs 多维度查询索引
-- 优化场景：按账号查询使用记录（用户仪表盘、账单统计）
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_usage_logs_account_time
ON usage_logs(account_id, created_at DESC);

-- 2. usage_logs 按模型统计索引
-- 优化场景：按模型聚合统计（运营分析、成本核算）
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_usage_logs_model_time
ON usage_logs(model, created_at DESC);

-- 3. accounts 过期查询索引
-- 优化场景：定时任务扫描过期账号、过期提醒
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_accounts_expired
ON accounts(expires_at)
WHERE deleted_at IS NULL AND expires_at IS NOT NULL;

-- 4. account_groups 按组查询索引
-- 优化场景：按分组筛选账号（账号管理、调度器）
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_account_groups_group_created
ON account_groups(group_id, created_at DESC);

-- 5. proxy_subscriptions 刷新扫描索引
-- 优化场景：定时任务扫描待刷新订阅
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_proxy_subscription_sources_refresh
ON proxy_subscription_sources(enabled, last_refreshed_at)
WHERE deleted_at IS NULL AND enabled = true;
