# 智能路由运维手册

**版本**: v1.0  
**更新日期**: 2026-10-07  
**适用版本**: sync/upstream-v0.2.4-20260910+

---

## 1. 监控指标

### 1.1 指标端点

**URL**: `GET /api/v1/routing/metrics`  
**认证**: 需要管理员token

**返回示例**:
```json
{
  "decisions_total_auto": 1234,
  "decisions_total_price": 567,
  "decisions_succeeded": 1500,
  "decisions_failed": 200,
  "decisions_no_route": 50,
  "fallback_depth_0": 1200,
  "fallback_depth_1": 300,
  "rank_duration_avg_us": 15000,
  "audit_dropped_total": 5,
  "stats_slow_count": 12,
  "stats_timeout_count": 2
}
```

### 1.2 关键指标说明

| 指标 | 说明 | 正常范围 | 告警阈值 |
|------|------|----------|----------|
| `decisions_succeeded` | 成功决策数 | - | - |
| `decisions_failed` | 失败决策数 | <5% | >10% |
| `decisions_no_route` | 无路由数 | <1% | >5% |
| `fallback_depth_0` | 主组直接成功 | >90% | <70% |
| `fallback_depth_1+` | 需要切换组 | <10% | >30% |
| `rank_duration_avg_us` | 排序平均耗时(微秒) | <50000 (50ms) | >200000 (200ms) |
| `audit_dropped_total` | 审计队列丢弃数 | 0 | >100/小时 |
| `stats_slow_count` | 慢查询(>100ms) | <10/小时 | >100/小时 |
| `stats_timeout_count` | 查询超时(>250ms) | 0 | >10/小时 |

---

## 2. 日志监控

### 2.1 关键日志

```bash
# 慢查询
grep "smart_routing_stats_slow" app.log

# 查询超时
grep "smart_routing_stats_timeout" app.log

# 审计队列满载
grep "smart_routing_audit_queue_full" app.log

# 无路由
grep "no_available_route" app.log
```

### 2.2 日志字段

**慢查询日志**:
```json
{
  "level": "WARN",
  "msg": "smart_routing_stats_slow",
  "group_id": 123,
  "model": "gpt-4o",
  "duration_ms": 150
}
```

**审计丢弃日志**:
```json
{
  "level": "WARN",
  "msg": "smart_routing_audit_queue_full",
  "request_id": "abc123",
  "kind": "decision",
  "total_dropped": 42
}
```

---

## 3. 常见问题排查

### 3.1 慢查询率上升

**症状**: `stats_slow_count` 持续增长，`rank_duration_avg_us` >100ms

**排查步骤**:
1. 检查数据库负载
   ```bash
   # PostgreSQL
   SELECT * FROM pg_stat_activity WHERE state = 'active';
   ```

2. 检查 `custom_route_attempts` 表大小
   ```sql
   SELECT pg_size_pretty(pg_total_relation_size('custom_route_attempts'));
   ```

3. 检查索引是否存在
   ```sql
   \d custom_route_attempts
   -- 应该有 (group_id, requested_model, endpoint_type, stream, created_at) 索引
   ```

**解决方案**:
- 如果表过大(>10GB): 考虑缩短保留期（当前24小时）
- 如果索引缺失: 重新运行迁移
- 如果数据库整体慢: 扩容或优化查询

---

### 3.2 审计队列频繁满载

**症状**: `audit_dropped_total` 快速增长

**排查步骤**:
1. 查看丢弃速率
   ```bash
   # 每分钟丢弃数
   grep "audit_queue_full" app.log | grep -o "total_dropped\":[0-9]*" | \
     awk -F: '{print $2}' | sort -n | tail -10
   ```

2. 检查数据库写入延迟
   ```sql
   SELECT * FROM pg_stat_statements 
   WHERE query LIKE '%custom_route_decisions%' 
   ORDER BY mean_exec_time DESC;
   ```

**解决方案**:
- 如果数据库慢: 优化或扩容
- 如果流量暴增: 这是设计的降级行为，遥测丢弃不影响路由功能
- 如果持续丢弃: 考虑增加队列容量（需修改代码）

---

### 3.3 故障转移频繁

**症状**: `fallback_depth_1+` 占比>20%

**排查步骤**:
1. 查看哪些分组频繁失败
   ```sql
   SELECT selected_group_id, COUNT(*), AVG(fallback_depth)
   FROM custom_route_outcomes
   WHERE created_at > NOW() - INTERVAL '1 hour'
   GROUP BY selected_group_id
   ORDER BY AVG(fallback_depth) DESC;
   ```

2. 检查账号健康状态
   ```bash
   redis-cli KEYS "custom:route:account:v2:*"
   ```

**解决方案**:
- 如果某个分组频繁失败: 检查该分组的账号配置和健康状态
- 如果主分组不稳定: 建议用户调整主分组或候选组顺序
- 如果上游API波动: 这是智能路由设计的正常行为

---

### 3.4 无路由率上升

**症状**: `decisions_no_route` >5%

**排查步骤**:
1. 查看无路由原因
   ```sql
   SELECT decision_reason, COUNT(*)
   FROM custom_route_decisions
   WHERE selected_group_id IS NULL
     AND created_at > NOW() - INTERVAL '1 hour'
   GROUP BY decision_reason;
   ```

**常见原因**:
- `budget exhausted`: 12次尝试预算用完
- `all accounts cooling`: 所有账号都在冷却期
- `insufficient capacity`: 容量不足
- `platform mismatch`: 平台不匹配

**解决方案**:
- 预算耗尽: 检查是否所有分组都不可用，可能需要增加候选组
- 全部冷却: 上游大面积故障，等待恢复或增加健康账号
- 容量不足: 扩容或限制并发

---

## 4. 配置调优

### 4.1 探索率

**当前默认**: 1% (0.01)  
**推荐范围**: 0.5% - 1%

**调整场景**:
- 新分组刚加入: 临时提高到1%加速数据收集
- 生产稳定期: 降低到0.5%减少探索开销
- 策略为价格/速度/成功: 自动禁用探索

### 4.2 候选组数量

**限制**: 最多8个  
**推荐**: 3-5个

**原因**:
- 候选组越多，rank()查询开销越大
- 超过5个后边际收益递减
- 故障转移深度通常不超过2次

### 4.3 权重配置

**预设策略**:
- 自动: price=40%, speed=30%, success=30%
- 价格优先: price=70%, speed=15%, success=15%
- 速度优先: price=15%, speed=70%, success=15%
- 成功率优先: price=15%, speed=15%, success=70%

**自定义建议**:
- 价格敏感场景: price=60-80%
- 实时性要求高: speed=60-80%
- 稳定性优先: success=60-80%

---

## 5. 故障演练

### 5.1 Redis断连恢复

**模拟**:
```bash
# 断开Redis
systemctl stop redis

# 观察日志
tail -f app.log | grep "smart_routing"

# 恢复Redis
systemctl start redis
```

**预期行为**:
- 健康账号继续服务（本地降级）
- 熔断账号进入本地冷却
- Redis恢复后自动切回分布式状态

---

### 5.2 数据库慢查询

**模拟**:
```sql
-- 人为添加慢查询
SELECT pg_sleep(0.3);
```

**预期行为**:
- 超过250ms查询超时，返回空统计
- 超过100ms记录slow日志
- rank()使用冷启动基准继续

---

### 5.3 高并发压测

**工具**: Apache Bench, wrk, k6

**示例**:
```bash
# 1000 RPS，持续30秒
wrk -t 10 -c 100 -d 30s \
  -H "Authorization: Bearer $TOKEN" \
  https://api.example.com/v1/chat/completions
```

**观察指标**:
- `audit_dropped_total` 应保持为0（如队列足够）
- `rank_duration_avg_us` 应<50ms
- `fallback_depth_0` 应>90%

---

## 6. 维护任务

### 6.1 每日检查

- [ ] 检查 `audit_dropped_total` 增量
- [ ] 检查 `stats_timeout_count`
- [ ] 检查 `decisions_no_route` 比例

### 6.2 每周检查

- [ ] 检查 `custom_route_decisions` 表大小
- [ ] 检查慢查询日志聚合
- [ ] 检查故障转移分布

### 6.3 每月检查

- [ ] Review候选组配置合理性
- [ ] 分析长期趋势
- [ ] 评估是否需要调整保留策略

---

## 7. 联系支持

**紧急问题**:
- Slack: #smart-routing-ops
- 值班电话: [待补充]

**非紧急**:
- Jira: [项目链接]
- 文档: 本手册

---

**最后更新**: 2026-10-07  
**维护者**: [待指定]
