# 数据库设计与查询优化审查报告

## 执行摘要

本报告对 sub2api 项目的数据库设计、索引策略和查询性能进行了全面审查。项目使用 PostgreSQL 15+ 作为主数据库，采用 Ent ORM 框架进行数据访问，并在复杂查询场景下使用原生 SQL。

**总体评估**: 数据库设计整体合理，但存在一些性能优化空间和潜在的查询瓶颈。

---

## 1. 数据库设计审查

### 1.1 Schema 设计评估

#### ✅ 优点

1. **清晰的表结构层次**
   - 核心表设计合理：users、accounts、api_keys、usage_logs
   - 关联表设计规范：account_groups、user_allowed_groups
   - 软删除机制完善：所有主表都有 deleted_at 字段

2. **时间戳管理**
   - 统一使用 TIMESTAMPTZ 类型，支持时区
   - created_at、updated_at 字段完整
   - 业务时间字段（last_used_at、expires_at）设计合理

3. **JSONB 字段使用**
   - credentials、extra 字段使用 JSONB 存储灵活数据
   - 适合存储非结构化配置和扩展信息

#### ⚠️ 问题与建议

**问题 1: usage_logs 表数据量增长**
- **位置**: `backend/migrations/001_init.sql:134-173`
- **描述**: usage_logs 表是核心日志表，随时间快速增长
- **影响**: 查询性能下降、存储成本增加
- **当前缓解措施**: 
  - 已实现分区表支持（migration 035）
  - 已实现清理任务（migration 042）
- **建议**: 
  - 考虑按月自动分区（已部分实现）
  - 定期归档历史数据到冷存储
  - 监控分区表性能

**问题 2: 软删除索引优化不足**
- **位置**: 多个表的 deleted_at 索引
- **描述**: 部分表的软删除索引未使用部分索引
- **影响**: 索引包含大量已删除记录，降低查询效率
- **建议**: 
  ```sql
  -- 优化示例：只索引未删除记录
  CREATE INDEX idx_api_keys_key_active 
    ON api_keys(key) 
    WHERE deleted_at IS NULL;
  ```

**问题 3: JSONB 字段缺少 GIN 索引**
- **位置**: accounts.credentials, accounts.extra
- **描述**: JSONB 字段查询时可能需要全表扫描
- **影响**: 按 credentials 内容查询时性能差
- **建议**: 
  ```sql
  -- 如果需要按 credentials 内容查询
  CREATE INDEX idx_accounts_credentials_gin 
    ON accounts USING GIN (credentials);
  ```

### 1.2 关系设计评估

#### ✅ 优点

1. **外键约束完整**
   - 主要关联都有外键约束
   - ON DELETE CASCADE/SET NULL 使用合理

2. **多对多关系处理**
   - account_groups 关联表设计规范
   - user_allowed_groups 支持灵活的权限控制

#### ⚠️ 问题

**问题 4: 潜在的循环依赖**
- **位置**: groups.fallback_group_id
- **描述**: 分组可以指向另一个分组作为 fallback
- **风险**: 可能形成循环引用
- **建议**: 在应用层添加循环检测逻辑

---

## 2. 索引使用审查

### 2.1 现有索引分析

项目共有 **188 个 migration 文件**，索引策略经过多次迭代优化。

#### ✅ 良好的索引设计

1. **复合索引优化**（migration 010, 062）
   ```sql
   -- 支持按账号+时间范围查询
   CREATE INDEX idx_usage_logs_account_created_at 
     ON usage_logs(account_id, created_at);
   
   -- 支持按 API Key + 时间范围查询
   CREATE INDEX idx_usage_logs_api_key_created_at 
     ON usage_logs(api_key_id, created_at);
   ```

2. **部分索引优化**（migration 062）
   ```sql
   -- 只索引活跃可调度账号
   CREATE INDEX idx_accounts_schedulable_hot
     ON accounts (platform, priority)
     WHERE deleted_at IS NULL 
       AND status = 'active' 
       AND schedulable = true;
   ```

3. **唯一索引与软删除**（migration 016）
   ```sql
   -- 软删除场景下的唯一约束
   CREATE UNIQUE INDEX users_email_unique_active
     ON users(email) 
     WHERE deleted_at IS NULL;
   ```

### 2.2 缺失的索引

#### 🔴 高优先级

**缺失索引 1: usage_logs 多维度查询**
- **查询场景**: 按用户+分组+时间范围统计
- **当前问题**: 可能需要扫描大量数据
- **建议**:
  ```sql
  CREATE INDEX idx_usage_logs_user_group_created 
    ON usage_logs(user_id, group_id, created_at)
    WHERE actual_cost > 0;  -- 只索引成功请求
  ```

**缺失索引 2: accounts 状态查询**
- **查询场景**: 查找过期或限流的账号
- **建议**:
  ```sql
  CREATE INDEX idx_accounts_expires_at 
    ON accounts(expires_at)
    WHERE expires_at IS NOT NULL 
      AND deleted_at IS NULL;
  
  CREATE INDEX idx_accounts_rate_limited 
    ON accounts(rate_limited_at, rate_limit_reset_at)
    WHERE rate_limited_at IS NOT NULL 
      AND deleted_at IS NULL;
  ```

**缺失索引 3: user_subscriptions 过期查询**
- **查询场景**: 定期清理过期订阅
- **建议**:
  ```sql
  CREATE INDEX idx_user_subscriptions_expires_status 
    ON user_subscriptions(expires_at, status)
    WHERE deleted_at IS NULL;
  ```

#### 🟡 中优先级

**缺失索引 4: api_keys 配额查询**
- **查询场景**: 查找配额即将用尽的 API Key
- **建议**:
  ```sql
  CREATE INDEX idx_api_keys_quota_usage 
    ON api_keys(quota, quota_used)
    WHERE deleted_at IS NULL 
      AND status = 'active' 
      AND quota > 0;
  ```

### 2.3 冗余索引

**冗余索引 1: 单列索引 vs 复合索引**
- **位置**: usage_logs 表
- **问题**: 
  - 存在 `idx_usage_logs_user_id`
  - 也存在 `idx_usage_logs_user_created`
- **说明**: 复合索引可以覆盖单列查询，但保留单列索引可能是为了特定查询优化
- **建议**: 通过 EXPLAIN ANALYZE 验证是否真正需要两个索引

---

## 3. 查询性能审查

### 3.1 N+1 查询问题

#### ✅ 已优化的场景

1. **API Key 认证查询**（api_key_repo.go:121-197）
   - 使用 `WithUser()` 和 `WithGroup()` 预加载关联数据
   - 使用 `Select()` 只查询必要字段
   - 避免了 N+1 查询

2. **账号批量查询**（account_repo.go:187-200）
   - `GetByIDs` 方法去重并批量查询
   - 使用 `accountsToService` 统一转换

#### 🔴 潜在的 N+1 问题

**N+1 问题 1: 用户允许分组加载**
- **位置**: `user_repo.go:126-133`
- **代码**:
  ```go
  groups, err := r.loadAllowedGroups(ctx, []int64{id})
  ```
- **问题**: 每次查询用户都单独加载 allowed_groups
- **影响**: 批量查询用户时会产生 N+1
- **建议**: 
  ```go
  // 批量加载所有用户的 allowed_groups
  func (r *userRepository) GetByIDs(ctx context.Context, ids []int64) ([]*service.User, error) {
      users, err := r.client.User.Query().Where(user.IDIn(ids...)).All(ctx)
      // ...
      allGroups, err := r.loadAllowedGroups(ctx, ids) // 一次查询所有
      for _, u := range users {
          if groups, ok := allGroups[u.ID]; ok {
              u.AllowedGroups = groups
          }
      }
  }
  ```

**N+1 问题 2: 账号分组关联**
- **位置**: `account_repo.go` 的 `accountsToService` 方法
- **问题**: 可能为每个账号单独查询分组关联
- **建议**: 使用 IN 查询批量加载

### 3.2 慢查询识别

#### 🔴 潜在慢查询

**慢查询 1: usage_logs 聚合统计**
- **位置**: `usage_log_repo.go` 多个聚合方法
- **查询特征**:
  ```sql
  SELECT user_id, SUM(actual_cost), COUNT(*)
  FROM usage_logs
  WHERE created_at >= $1 AND created_at < $2
  GROUP BY user_id
  ```
- **问题**: 
  - 大时间范围查询会扫描大量数据
  - 多维度 GROUP BY 性能差
- **当前缓解**: 
  - 已实现预聚合表（usage_dashboard_hourly/daily）
  - 使用 Redis 缓存
- **建议**: 
  - 强制使用预聚合表查询历史数据
  - 只允许查询最近 24 小时的原始数据

**慢查询 2: 账号调度查询**
- **位置**: 账号调度逻辑
- **查询特征**: 查找可用账号（多条件过滤）
- **优化**: 已通过部分索引优化（idx_accounts_schedulable_hot）
- **建议**: 监控查询性能，考虑使用 Redis 缓存热点账号

**慢查询 3: 软删除记录清理**
- **位置**: 各表的软删除记录
- **问题**: 软删除记录累积影响查询性能
- **建议**: 
  - 定期物理删除旧的软删除记录
  - 或将软删除记录移到归档表

### 3.3 批量操作优化

#### ✅ 良好实践

1. **批量插入优化**（usage_log_repo.go:189-199）
   - 使用批量插入队列
   - 批量大小：64 条
   - 批量窗口：3ms

2. **批量清理优化**（dashboard_aggregation_repo.go:224-245）
   - 使用 CTE 限制批量大小
   - 批量大小：10000 条
   - 避免长事务锁表

---

## 4. 事务使用审查

### 4.1 事务设计

#### ✅ 良好实践

1. **嵌套事务处理**（user_repo.go:42-67）
   - 检测已存在的事务（ErrTxStarted）
   - 避免重复开启事务
   - 正确处理事务提交/回滚

2. **事务隔离**
   - 使用 PostgreSQL 默认隔离级别（Read Committed）
   - 关键操作使用行锁（pg_advisory_lock）

#### ⚠️ 问题

**问题 5: 长事务风险**
- **位置**: 批量操作和聚合任务
- **问题**: 长事务可能导致锁等待和连接池耗尽
- **建议**: 
  - 限制单个事务的操作数量
  - 使用批量提交策略
  - 监控事务持续时间

---

## 5. 连接池配置审查

### 5.1 当前配置

**数据库连接池**（config.go:1044-1059）
```yaml
database:
  max_open_conns: 256      # 最大连接数
  max_idle_conns: 128      # 最大空闲连接
  conn_max_lifetime_minutes: 30
  conn_max_idle_time_minutes: 5
```

**Redis 连接池**（config.go:1093-1113）
```yaml
redis:
  pool_size: 1024          # 连接池大小
  min_idle_conns: 128      # 最小空闲连接
  dial_timeout_seconds: 5
  read_timeout_seconds: 3
  write_timeout_seconds: 3
```

### 5.2 评估

#### ✅ 优点
- 连接池参数可配置
- 连接生命周期管理合理
- 超时设置完善

#### ⚠️ 建议

**建议 1: 动态调整连接池**
- 根据实际负载监控调整
- 高峰期可能需要更多连接
- 建议监控指标：
  - 连接池使用率
  - 等待连接的请求数
  - 连接获取延迟

**建议 2: 连接池预热**
- 启动时预创建最小连接数
- 避免冷启动时的连接延迟

---

## 6. 查询超时设置

### 6.1 当前配置

**网关超时**（config.go:674）
```yaml
gateway:
  response_header_timeout: 600  # 10分钟
```

### 6.2 建议

**建议 1: 设置语句超时**
```sql
-- 在连接级别设置
SET statement_timeout = '30s';

-- 或在应用层设置
ALTER DATABASE sub2api SET statement_timeout = '30s';
```

**建议 2: 区分查询类型**
- 在线查询：5-10 秒超时
- 批量操作：30-60 秒超时
- 后台任务：可以更长

---

## 7. 优化建议总结

### 7.1 高优先级（立即执行）

1. **添加缺失的复合索引**
   - usage_logs(user_id, group_id, created_at)
   - accounts(expires_at)
   - user_subscriptions(expires_at, status)

2. **修复 N+1 查询问题**
   - 优化 user allowed_groups 加载
   - 批量加载账号分组关联

3. **设置查询超时**
   - 数据库级别 statement_timeout
   - 应用级别 context timeout

### 7.2 中优先级（1-2 周内）

4. **优化软删除索引**
   - 将常用查询索引改为部分索引
   - 只索引未删除记录

5. **监控慢查询**
   - 启用 pg_stat_statements
   - 设置慢查询日志阈值
   - 定期分析慢查询

6. **优化批量操作**
   - 审查长事务
   - 优化批量大小

### 7.3 低优先级（长期优化）

7. **数据归档策略**
   - 定期归档历史 usage_logs
   - 清理旧的软删除记录

8. **读写分离**
   - 考虑使用只读副本
   - 统计查询路由到副本

9. **缓存优化**
   - 扩展 Redis 缓存使用
   - 实现查询结果缓存

---

## 8. 监控建议

### 8.1 关键指标

1. **查询性能**
   - 慢查询数量和占比
   - 平均查询时间
   - P95/P99 查询延迟

2. **连接池**
   - 连接池使用率
   - 等待连接的请求数
   - 连接获取延迟

3. **索引使用**
   - 索引命中率
   - 未使用的索引
   - 索引膨胀

4. **表大小**
   - 各表行数和大小
   - 增长趋势
   - 分区表分布

### 8.2 监控工具

推荐使用：
- **pg_stat_statements**: 查询统计
- **pgBadger**: 日志分析
- **Prometheus + Grafana**: 指标监控
- **项目内置 ops 监控**: 已实现部分监控功能

---

## 9. 具体优化 SQL

### 9.1 立即执行的索引创建

```sql
-- 1. usage_logs 多维度查询优化
CREATE INDEX CONCURRENTLY idx_usage_logs_user_group_created 
  ON usage_logs(user_id, group_id, created_at)
  WHERE actual_cost > 0 AND deleted_at IS NULL;

-- 2. accounts 过期查询优化
CREATE INDEX CONCURRENTLY idx_accounts_expires_at 
  ON accounts(expires_at)
  WHERE expires_at IS NOT NULL AND deleted_at IS NULL;

-- 3. accounts 限流查询优化
CREATE INDEX CONCURRENTLY idx_accounts_rate_limited 
  ON accounts(rate_limited_at, rate_limit_reset_at)
  WHERE rate_limited_at IS NOT NULL AND deleted_at IS NULL;

-- 4. user_subscriptions 过期查询优化
CREATE INDEX CONCURRENTLY idx_user_subscriptions_expires_status 
  ON user_subscriptions(expires_at, status)
  WHERE deleted_at IS NULL;

-- 5. api_keys 配额查询优化
CREATE INDEX CONCURRENTLY idx_api_keys_quota_usage 
  ON api_keys(quota, quota_used)
  WHERE deleted_at IS NULL 
    AND status = 'active' 
    AND quota > 0;
```

### 9.2 查询超时设置

```sql
-- 设置数据库级别超时
ALTER DATABASE sub2api SET statement_timeout = '30s';
ALTER DATABASE sub2api SET idle_in_transaction_session_timeout = '60s';

-- 为特定用户设置不同超时
ALTER ROLE sub2api_app SET statement_timeout = '10s';
ALTER ROLE sub2api_batch SET statement_timeout = '5min';
```

### 9.3 监控查询

```sql
-- 启用 pg_stat_statements
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;

-- 查看慢查询
SELECT 
  query,
  calls,
  total_exec_time,
  mean_exec_time,
  max_exec_time
FROM pg_stat_statements
WHERE mean_exec_time > 100  -- 超过 100ms
ORDER BY total_exec_time DESC
LIMIT 20;

-- 查看未使用的索引
SELECT 
  schemaname,
  tablename,
  indexname,
  idx_scan,
  pg_size_pretty(pg_relation_size(indexrelid)) AS index_size
FROM pg_stat_user_indexes
WHERE idx_scan = 0
  AND indexrelname NOT LIKE '%_pkey'
ORDER BY pg_relation_size(indexrelid) DESC;

-- 查看表膨胀
SELECT 
  schemaname,
  tablename,
  pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS total_size,
  pg_size_pretty(pg_relation_size(schemaname||'.'||tablename)) AS table_size,
  pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename) - pg_relation_size(schemaname||'.'||tablename)) AS indexes_size
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC
LIMIT 20;
```

---

## 10. 总结

### 10.1 整体评价

sub2api 项目的数据库设计整体上是**良好的**，具有以下优点：

✅ **优点**:
- Schema 设计清晰合理
- 索引策略经过多次优化
- 软删除机制完善
- 批量操作优化到位
- 连接池配置合理
- 已实现预聚合和缓存

⚠️ **需要改进**:
- 存在部分 N+1 查询问题
- 缺少一些关键索引
- 慢查询监控不足
- 查询超时未设置
- 软删除记录清理策略需完善

### 10.2 优先级矩阵

| 优先级 | 问题 | 影响 | 工作量 |
|--------|------|------|--------|
| 🔴 高 | 添加缺失索引 | 高 | 低 |
| 🔴 高 | 修复 N+1 查询 | 高 | 中 |
| 🔴 高 | 设置查询超时 | 高 | 低 |
| 🟡 中 | 优化软删除索引 | 中 | 中 |
| 🟡 中 | 监控慢查询 | 中 | 中 |
| 🟢 低 | 数据归档策略 | 低 | 高 |
| 🟢 低 | 读写分离 | 低 | 高 |

### 10.3 预期收益

实施上述优化后，预期可以获得：

1. **查询性能提升 30-50%**（通过添加索引和修复 N+1）
2. **减少慢查询 60-80%**（通过索引优化和查询超时）
3. **降低数据库负载 20-30%**（通过缓存和预聚合）
4. **提高系统稳定性**（通过超时设置和连接池优化）

---

## 附录

### A. 相关文件清单

**Schema 定义**:
- `backend/migrations/001_init.sql` - 初始 schema
- `backend/migrations/003_subscription.sql` - 订阅相关表
- `backend/migrations/010_add_usage_logs_aggregated_indexes.sql` - 聚合索引
- `backend/migrations/062_add_scheduler_and_usage_composite_indexes_notx.sql` - 复合索引

**Repository 实现**:
- `backend/internal/repository/user_repo.go` - 用户仓储
- `backend/internal/repository/account_repo.go` - 账号仓储
- `backend/internal/repository/api_key_repo.go` - API Key 仓储
- `backend/internal/repository/usage_log_repo.go` - 使用日志仓储
- `backend/internal/repository/dashboard_aggregation_repo.go` - 聚合仓储

**配置文件**:
- `backend/internal/config/config.go` - 数据库和 Redis 配置

### B. 审查方法

本次审查采用以下方法：

1. **静态代码分析**: 审查 migration 文件和 repository 代码
2. **索引分析**: 检查所有 CREATE INDEX 语句
3. **查询模式分析**: 识别常见查询模式和潜在问题
4. **配置审查**: 检查连接池和超时配置

### C. 后续行动

1. **立即执行**（本周）:
   - 创建缺失的索引
   - 设置查询超时
   - 启用慢查询日志

2. **短期执行**（1-2 周）:
   - 修复 N+1 查询问题
   - 优化软删除索引
   - 部署监控工具

3. **长期规划**（1-3 月）:
   - 实施数据归档策略
   - 评估读写分离方案
   - 持续优化查询性能

---

**报告生成时间**: 2026-05-21  
**审查范围**: 数据库 schema、索引、查询、事务、连接池  
**审查方法**: 静态代码分析 + 最佳实践对比
