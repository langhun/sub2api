# 智能路由修复快速参考

## ✅ 已修复 (7项)

### 🔴 P1 - 用户体验 (3项)

1. **Checkbox状态Bug** → `RoutingCandidateOptions.vue`
   - 添加 `selectedValues` prop
   - 实现 `isSelected()` 函数
   - 绑定到 `:checked`

2. **版本冲突处理** → `api.ts`
   - 新增 `VersionConflictError` 类
   - `updateSmartRouting()` 捕获400错误
   - 返回友好错误消息

3. **慢查询监控** → `module.go`
   - `loadStats()` 添加计时
   - 区分超时 vs 其他错误
   - 100ms阈值慢查询警告

### 🟡 P2 - 代码质量 (4项)

4. **Magic Numbers** → `module.go`
   - 提取8个常量
   - 统一超时、阈值配置

5. **审计丢弃计数** → `feedback.go`, `audit.go`
   - 添加 `auditDroppedCount atomic.Int64`
   - 日志输出累计丢弃数

6. **探索率常量** → `module.go`
   - `defaultExplorationRateMax = 0.01`
   - 所有引用统一

7. **前端常量导出** → `api.ts`
   - `export const MAX_EXPLORATION_RATE = 0.01`

---

## 📋 测试结果

- ✅ 后端: 所有测试通过 (0.341s)
- ✅ 前端: TypeScript检查通过
- ✅ 无回归
- ✅ 向后兼容

---

## 📦 部署清单

### 文件变更 (5个)
```
backend/internal/custom/modules/smart-routing/
  ├── module.go          (+50, -10)
  ├── feedback.go        (+2, -1)
  └── audit.go          (+3, -2)

frontend/src/custom/modules/smart-routing/
  ├── api.ts                          (+25, -3)
  └── RoutingCandidateOptions.vue    (+7, -3)
```

### 新增日志
- `smart_routing_stats_slow` - 慢查询（>100ms）
- `smart_routing_stats_timeout` - 查询超时
- `total_dropped` - 累计丢弃审计事件

### 监控建议
```bash
# 查看慢查询
grep "smart_routing_stats_slow" app.log | tail -20

# 查看审计丢弃
grep "audit_queue_full" app.log | grep "total_dropped"

# 统计超时率
grep "smart_routing_stats" app.log | \
  awk '{if($0~/timeout/) timeout++; total++} END {print timeout/total*100"%"}'
```

---

## 🚀 灰度方案

### Week 1: 测试环境
- 验证checkbox状态
- 触发版本冲突测试
- 观察慢查询日志

### Week 2: 10%流量
- 监控 `stats_slow` 指标
- 监控 `audit_dropped` 计数
- 用户反馈收集

### Week 3: 全量上线
- 持续监控1周
- 无问题则标记完成

---

## 🔄 回滚计划

| 组件 | 回滚难度 | 影响 |
|------|---------|------|
| 前端 Checkbox | 简单 | 仅UI显示 |
| 版本冲突处理 | 简单 | 错误消息 |
| 慢查询日志 | 无需回滚 | 仅日志 |
| 常量提取 | 无需回滚 | 无行为变化 |

**一键回滚命令**:
```bash
git revert <commit-hash>
```

---

## 📞 联系人

- **代码Review**: [指派给资深开发]
- **测试验证**: [指派给QA]
- **上线执行**: [指派给DevOps]
- **监控值班**: [指派给SRE]

---

**创建日期**: 2026-10-07  
**预计上线**: 2026-10-21 (2周后)  
**风险等级**: 🟢 低
