package smartrouting

import (
	"sync/atomic"
)

// MetricsSnapshot 是智能路由的运行时指标快照
type MetricsSnapshot struct {
	// 决策指标
	DecisionsTotalAuto    int64 `json:"decisions_total_auto"`
	DecisionsTotalPrice   int64 `json:"decisions_total_price"`
	DecisionsTotalSpeed   int64 `json:"decisions_total_speed"`
	DecisionsTotalSuccess int64 `json:"decisions_total_success"`
	DecisionsTotalCustom  int64 `json:"decisions_total_custom"`

	// 结果分布
	DecisionsSucceeded int64 `json:"decisions_succeeded"`
	DecisionsFailed    int64 `json:"decisions_failed"`
	DecisionsNoRoute   int64 `json:"decisions_no_route"`

	// 故障转移深度分布
	FallbackDepth0 int64 `json:"fallback_depth_0"` // 主组直接成功
	FallbackDepth1 int64 `json:"fallback_depth_1"` // 切换1次
	FallbackDepth2 int64 `json:"fallback_depth_2"` // 切换2次
	FallbackDepth3Plus int64 `json:"fallback_depth_3_plus"` // 切换3次及以上

	// 排序性能（微秒）
	RankDurationAvgUs int64 `json:"rank_duration_avg_us"`
	RankDurationMaxUs int64 `json:"rank_duration_max_us"`
	RankSampleCount   int64 `json:"rank_sample_count"`

	// 审计队列
	AuditDroppedTotal int64 `json:"audit_dropped_total"`

	// 慢查询统计
	StatsSlowCount    int64 `json:"stats_slow_count"`
	StatsTimeoutCount int64 `json:"stats_timeout_count"`
}

// routingMetrics 持有所有智能路由指标计数器
type routingMetrics struct {
	// 决策计数（按策略）
	decisionsAuto    atomic.Int64
	decisionsPrice   atomic.Int64
	decisionsSpeed   atomic.Int64
	decisionsSuccess atomic.Int64
	decisionsCustom  atomic.Int64

	// 结果计数
	decisionsSucceeded atomic.Int64
	decisionsFailed    atomic.Int64
	decisionsNoRoute   atomic.Int64

	// 故障转移深度
	fallbackDepth0     atomic.Int64
	fallbackDepth1     atomic.Int64
	fallbackDepth2     atomic.Int64
	fallbackDepth3Plus atomic.Int64

	// 排序性能追踪
	rankDurationSum   atomic.Int64 // 微秒累计
	rankDurationMax   atomic.Int64 // 微秒最大值
	rankSampleCount   atomic.Int64

	// 审计丢弃（从feedback引用）
	auditDroppedCount *atomic.Int64

	// 慢查询统计
	statsSlowCount    atomic.Int64
	statsTimeoutCount atomic.Int64
}

// recordDecisionByStrategy 记录决策（按策略分类）
func (m *routingMetrics) recordDecisionByStrategy(strategy string) {
	switch strategy {
	case "auto":
		m.decisionsAuto.Add(1)
	case "price":
		m.decisionsPrice.Add(1)
	case "speed":
		m.decisionsSpeed.Add(1)
	case "success":
		m.decisionsSuccess.Add(1)
	case "custom":
		m.decisionsCustom.Add(1)
	}
}

// recordDecisionOutcome 记录决策结果
func (m *routingMetrics) recordDecisionOutcome(succeeded bool, noRoute bool) {
	if noRoute {
		m.decisionsNoRoute.Add(1)
		return
	}
	if succeeded {
		m.decisionsSucceeded.Add(1)
	} else {
		m.decisionsFailed.Add(1)
	}
}

// recordFallbackDepth 记录故障转移深度
func (m *routingMetrics) recordFallbackDepth(depth int) {
	switch {
	case depth == 0:
		m.fallbackDepth0.Add(1)
	case depth == 1:
		m.fallbackDepth1.Add(1)
	case depth == 2:
		m.fallbackDepth2.Add(1)
	default:
		m.fallbackDepth3Plus.Add(1)
	}
}

// recordRankDuration 记录排序耗时（微秒）
func (m *routingMetrics) recordRankDuration(durationUs int64) {
	m.rankDurationSum.Add(durationUs)
	m.rankSampleCount.Add(1)

	// 更新最大值（简单的原子读-写，可能有轻微竞争但不影响监控准确性）
	for {
		currentMax := m.rankDurationMax.Load()
		if durationUs <= currentMax {
			break
		}
		if m.rankDurationMax.CompareAndSwap(currentMax, durationUs) {
			break
		}
	}
}

// recordStatsSlowQuery 记录慢查询
func (m *routingMetrics) recordStatsSlowQuery() {
	m.statsSlowCount.Add(1)
}

// recordStatsTimeout 记录查询超时
func (m *routingMetrics) recordStatsTimeout() {
	m.statsTimeoutCount.Add(1)
}

// Snapshot 返回当前指标快照
func (m *routingMetrics) Snapshot() MetricsSnapshot {
	avgUs := int64(0)
	if count := m.rankSampleCount.Load(); count > 0 {
		avgUs = m.rankDurationSum.Load() / count
	}

	auditDropped := int64(0)
	if m.auditDroppedCount != nil {
		auditDropped = m.auditDroppedCount.Load()
	}

	return MetricsSnapshot{
		DecisionsTotalAuto:    m.decisionsAuto.Load(),
		DecisionsTotalPrice:   m.decisionsPrice.Load(),
		DecisionsTotalSpeed:   m.decisionsSpeed.Load(),
		DecisionsTotalSuccess: m.decisionsSuccess.Load(),
		DecisionsTotalCustom:  m.decisionsCustom.Load(),

		DecisionsSucceeded: m.decisionsSucceeded.Load(),
		DecisionsFailed:    m.decisionsFailed.Load(),
		DecisionsNoRoute:   m.decisionsNoRoute.Load(),

		FallbackDepth0:     m.fallbackDepth0.Load(),
		FallbackDepth1:     m.fallbackDepth1.Load(),
		FallbackDepth2:     m.fallbackDepth2.Load(),
		FallbackDepth3Plus: m.fallbackDepth3Plus.Load(),

		RankDurationAvgUs: avgUs,
		RankDurationMaxUs: m.rankDurationMax.Load(),
		RankSampleCount:   m.rankSampleCount.Load(),

		AuditDroppedTotal: auditDropped,

		StatsSlowCount:    m.statsSlowCount.Load(),
		StatsTimeoutCount: m.statsTimeoutCount.Load(),
	}
}

// SnapshotMetrics 返回智能路由指标快照（导出供HTTP handler使用）
func (m *Module) SnapshotMetrics() MetricsSnapshot {
	if m == nil || m.metrics == nil {
		return MetricsSnapshot{}
	}
	return m.metrics.Snapshot()
}
