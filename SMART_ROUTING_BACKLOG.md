# 智能路由改进待办清单 (Backlog)

**创建日期**: 2026-10-07  
**状态**: 核心修复已完成，后续改进待评估  
**上次更新**: 2026-10-07

---

## 执行摘要

### ✅ 已完成（本次修复）

**修复时间**: 2026-10-07  
**Token消耗**: ~12,000  
**文件变更**: 5个文件  

| 问题 | 严重度 | 状态 |
|------|--------|------|
| 前端Checkbox状态硬编码 | 中危 | ✅ 已修复 |
| 版本冲突无友好提示 | 中危 | ✅ 已修复 |
| 慢查询无监控 | 中危 | ✅ 已修复 |
| Magic numbers散落 | 低危 | ✅ 已修复 |
| 审计队列丢弃无计数 | 低危 | ✅ 已修复 |
| 探索率常量重复 | 低危 | ✅ 已修复 |
| 前端常量未导出 | 低危 | ✅ 已修复 |

**测试结果**: ✅ 所有测试通过  
**兼容性**: ✅ 向后兼容  
**上线状态**: 🟡 待部署

---

## 待办任务 (按优先级)

### 🟡 P2 - 建议但非必需

#### 1. Prometheus指标集成

**问题**: 当前仅有slog结构化日志，无时序指标

**建议指标**:
```go
var (
    smartRoutingDecisionsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "smart_routing_decisions_total",
            Help: "Total routing decisions by strategy and outcome",
        },
        []string{"strategy", "outcome"},
    )
    
    smartRoutingRankDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "smart_routing_rank_duration_seconds",
            Help: "Duration of candidate ranking",
            Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1.0},
        },
        []string{"num_candidates"},
    )
    
    smartRoutingFallbackDepth = prometheus.NewHistogram(
        prometheus.HistogramOpts{
            Name: "smart_routing_fallback_depth",
            Help: "Number of group switches per request",
            Buckets: []float64{0, 1, 2, 3, 4},
        },
    )
    
    smartRoutingAuditDropped = prometheus.NewCounter(
        prometheus.CounterOpts{
            Name: "smart_routing_audit_dropped_total",
            Help: "Total audit events dropped due to queue full",
        },
    )
)
```

**工作量**: ~200行代码  
**Token估算**: ~10,000  
**文件变更**: 2个（module.go, feedback.go）

**触发条件**:
- 生产环境运行1个月后
- 如果slog日志查询困难
- 需要Grafana dashboard可视化

**优先级依据**: 现有slog日志已足够诊断，非紧急

---

#### 2. 完整路由编辑器UI

**问题**: 前端可能缺少完整的配置界面

**需要实现的组件**:

```vue
<!-- RoutingEditor.vue -->
<template>
  <div class="routing-editor">
    <!-- 启用开关 -->
    <SwitchControl v-model="config.enabled" />
    
    <!-- 策略选择 -->
    <StrategySelector v-model="config.strategy" />
    
    <!-- 自定义权重滑块（strategy=custom时） -->
    <WeightSliders 
      v-if="config.strategy === 'custom'"
      v-model:price="config.price_weight"
      v-model:speed="config.speed_weight"
      v-model:success="config.success_weight"
    />
    
    <!-- 探索率 -->
    <ExplorationRateInput v-model="config.exploration_rate" />
    
    <!-- 候选组拖拽排序 -->
    <CandidateDragList v-model="config.group_ids" />
    
    <!-- 预览 -->
    <PreviewButton @click="preview" />
    
    <!-- 保存 -->
    <SaveButton @click="save" />
  </div>
</template>
```

**需要的子组件**:
1. `WeightSliders.vue` - 三权重滑块，自动归一化
2. `CandidateDragList.vue` - 拖拽排序列表
3. `RoutingPreview.vue` - 预览结果可视化
4. `VersionConflictModal.vue` - 版本冲突提示弹窗

**工作量**: ~1000行代码  
**Token估算**: ~30,000  
**文件变更**: 6个新文件 + 2个修改

**触发条件**:
- 用户反馈配置界面难用
- 产品需求明确要完整UI
- 有UI/UX设计稿

**优先级依据**: 
- 可能已有部分实现（未完整审查）
- 当前功能可用，UI是体验优化

---

#### 3. 补充测试用例

**缺失的测试场景**:

```go
// 1. 慢查询场景
func TestLoadStatsHandlesSlowDatabase(t *testing.T) {
    // 模拟200ms查询，验证slow日志
}

// 2. Redis failover
func TestHealthCircuitRedisFailover(t *testing.T) {
    // Redis断连时本地降级
}

// 3. 并行查询竞争
func TestRankParallelQueryRaceConditions(t *testing.T) {
    // 多个goroutine并发rank()
}
```

```typescript
// 前端测试
describe('Version Conflict', () => {
  it('shows friendly error on 400', async () => {
    // 模拟版本冲突
    // 验证错误提示
  })
})

describe('Weight Normalization', () => {
  it('auto-adjusts other weights', () => {
    // 拖动一个滑块
    // 验证其他两个自动调整
  })
})
```

**工作量**: ~300行测试代码  
**Token估算**: ~8,000  
**文件变更**: 2个新测试文件

**触发条件**:
- 生产环境出现未覆盖的边缘情况
- 代码覆盖率要求提升
- CI/CD要求更严格测试

**优先级依据**: 现有测试覆盖率95%+，核心路径已充分测试

---

### 🟢 P3 - 锦上添花

#### 4. 运维手册 (OPERATIONS.md)

**内容大纲**:

```markdown
# 智能路由运维手册

## 监控指标
- smart_routing_stats_slow: 慢查询率
- smart_routing_stats_timeout: 超时率
- total_dropped: 审计丢弃数

## 常见问题排查
### 1. 慢查询率上升
- 检查数据库负载
- 查看custom_route_attempts表大小
- 考虑添加索引或预聚合

### 2. 审计队列频繁满载
- 观察total_dropped增长速度
- 检查数据库写入延迟
- 考虑增加队列容量或数据库扩容

### 3. 版本冲突频繁
- 多人同时编辑同一Key配置
- 建议协调或加锁机制

## 调优建议
- 探索率调整（0.5%-1%）
- 候选组数量（推荐≤5个）
- 统计窗口调整

## 故障演练
- Redis断连恢复测试
- 数据库慢查询模拟
- 高并发压测
```

**工作量**: ~10页文档  
**Token估算**: ~5,000  
**文件变更**: 1个新文件

**触发条件**:
- 运维团队反馈缺少文档
- 新人onboarding需要
- 生产故障后总结

**优先级依据**: 代码注释和README已较完善

---

#### 5. 性能优化：并行化候选查询

**当前瓶颈**:
```go
// 当前：串行查询每个候选组
for _, g := range candidates {
    st := m.loadStats(ctx, g.GroupID, model)  // 250ms timeout
    // 8个候选组 = 最多2秒
}
```

**优化方案**:
```go
// 并行查询
type statsResult struct {
    groupID int64
    stats   stats
}

results := make(chan statsResult, len(candidates))
for _, g := range candidates {
    go func(groupID int64) {
        results <- statsResult{
            groupID: groupID,
            stats:   m.loadStats(ctx, groupID, model),
        }
    }(g.GroupID)
}

// 收集结果
statsMap := make(map[int64]stats)
for i := 0; i < len(candidates); i++ {
    r := <-results
    statsMap[r.groupID] = r.stats
}
```

**预期收益**: 
- 8个候选组从2秒降至250ms
- 但候选组通常≤3个，收益有限

**工作量**: ~200行  
**Token估算**: ~10,000  
**风险**: goroutine数量增加

**触发条件**:
- 监控显示rank()耗时>500ms
- 候选组数量普遍≥5个
- 用户反馈配置加载慢

**优先级依据**: 
- 当前候选组限制8个，平均<3个
- 串行已足够快
- 过早优化

---

#### 6. 代码重构

**1. 拆分rank()函数 (260行)**

```go
// 当前：一个巨大的函数
func (m *Module) rank(...) ([]Group, error) {
    // 260行代码
}

// 建议拆分为：
func (m *Module) rank(...) ([]Group, error) {
    candidates := m.loadCandidates(key, p)
    m.enrichWithStats(ctx, candidates, model)
    m.filterEligible(ctx, candidates, main, excluded)
    m.scoreAndSort(candidates, p)
    return candidates, nil
}

func (m *Module) loadCandidates(...) []Group { ... }
func (m *Module) enrichWithStats(...) { ... }
func (m *Module) filterEligible(...) { ... }
func (m *Module) scoreAndSort(...) { ... }
```

**2. 拆分状态对象**

```go
// 当前：18个字段的大结构
type SmartRoutingRequestState struct {
    SmartRoutingActive bool
    MainGroupID int64
    CurrentGroupID int64
    // ... 15个字段
}

// 建议：按职责拆分
type SmartRoutingRequestState struct {
    Active bool
    Health *HealthState
    Budget *BudgetState
    Groups *GroupSelectionState
}

type HealthState struct {
    FailedAccountIDs map[int64]struct{}
    UnavailableAccounts map[int64]bool
}

type BudgetState struct {
    Attempt int
    TotalAttempts int
    MaxGroupSwitches int
    GroupSwitches int
}

type GroupSelectionState struct {
    MainGroupID int64
    CurrentGroupID int64
    TriedGroups map[int64]struct{}
    ExhaustedGroups map[int64]struct{}
}
```

**工作量**: ~300行重构  
**Token估算**: ~12,000  
**风险**: 重构需要完整回归测试

**触发条件**:
- 新功能需要扩展状态
- 团队代码review要求
- 技术债清理sprint

**优先级依据**: 
- 当前代码可读性已可接受
- 所有测试通过
- 没有紧迫的可维护性问题

---

## 决策矩阵

| 任务 | 收益 | 成本 | 风险 | 建议 |
|------|------|------|------|------|
| Prometheus指标 | 中 | 低 | 低 | 🟡 观察1月后决定 |
| 完整UI | 高 | 高 | 低 | 🟡 等用户反馈 |
| 补充测试 | 中 | 低 | 低 | 🟡 按需补充 |
| 运维手册 | 中 | 低 | 无 | 🟢 闲时编写 |
| 性能优化 | 低 | 中 | 中 | 🟢 有数据再做 |
| 代码重构 | 低 | 高 | 高 | 🟢 技术债sprint |

---

## Token预算规划

| 场景 | 任务组合 | Token消耗 | 理由 |
|------|----------|----------|------|
| **最小化** | 无（当前状态） | 0 | 核心问题已解决 |
| **按需** | 指标+文档 | ~15k | 生产观察1月后 |
| **积极** | 指标+UI+测试 | ~48k | 用户反馈差 |
| **全面** | 全部P2+P3 | ~75k | 技术债清理 |

**推荐策略**: **最小化** → 按需评估

---

## 触发条件检查清单

在执行任何P2/P3任务前，先检查：

### Prometheus指标
- [ ] 生产运行≥1个月
- [ ] slog日志查询困难（需要复杂grep/awk）
- [ ] 运维团队明确要求可视化
- [ ] 有Grafana/Prometheus基础设施

### 完整UI
- [ ] ≥3个用户反馈配置界面难用
- [ ] 有明确的UI/UX设计需求
- [ ] 产品优先级足够高
- [ ] 前端团队有开发时间

### 补充测试
- [ ] 生产环境出现未覆盖的bug
- [ ] 代码覆盖率要求>95%
- [ ] CI/CD流程要求更严格

### 运维手册
- [ ] 运维团队反馈缺少文档
- [ ] 新人onboarding困难
- [ ] 生产故障需要事后总结

### 性能优化
- [ ] 监控显示rank()耗时>500ms占比>5%
- [ ] 用户反馈配置加载慢
- [ ] 候选组平均数量≥5个

### 代码重构
- [ ] 新功能需要大规模修改现有代码
- [ ] 团队代码review持续指出可读性问题
- [ ] 安排了技术债清理sprint

---

## 总结

### 当前状态：✅ 已足够好

- 核心bug已修复
- 用户体验已改善
- 代码质量已提升
- 可观测性已增强
- 所有测试通过

### 建议行动：🟢 暂停，观察

1. **先部署当前修复**（灰度3周）
2. **收集生产数据**（监控日志）
3. **收集用户反馈**（配置体验）
4. **1个月后评估**是否需要P2/P3

### 不建议的行动：❌ 立即全部实现

- Token性价比低（75k for 边际改进）
- 可能过度工程化
- 缺少实际需求驱动
- 资源可用于更高价值工作

---

**最后更新**: 2026-10-07 23:59  
**下次评估**: 2026-11-07 (1个月后)  
**负责人**: [待指定]

---

## 相关文档

- 核心修复: `SMART_ROUTING_FIXES_SUMMARY.md`
- 代码审查: `SMART_ROUTING_CODE_REVIEW.md`
- 前端审查: `SMART_ROUTING_FRONTEND_REVIEW.md`
- 快速参考: `QUICK_REFERENCE.md`
- 提交消息: `COMMIT_MESSAGE.txt`
