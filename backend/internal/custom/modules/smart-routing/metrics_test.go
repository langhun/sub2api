package smartrouting

import (
	"testing"
)

func TestMetricsSnapshot(t *testing.T) {
	m := &routingMetrics{}

	// 记录一些指标
	m.recordDecisionByStrategy("auto")
	m.recordDecisionByStrategy("price")
	m.recordDecisionOutcome(true, false)
	m.recordFallbackDepth(0)
	m.recordFallbackDepth(1)
	m.recordRankDuration(1000) // 1ms = 1000us
	m.recordStatsSlowQuery()

	snapshot := m.Snapshot()

	if snapshot.DecisionsTotalAuto != 1 {
		t.Errorf("expected 1 auto decision, got %d", snapshot.DecisionsTotalAuto)
	}

	if snapshot.DecisionsTotalPrice != 1 {
		t.Errorf("expected 1 price decision, got %d", snapshot.DecisionsTotalPrice)
	}

	if snapshot.DecisionsSucceeded != 1 {
		t.Errorf("expected 1 succeeded decision, got %d", snapshot.DecisionsSucceeded)
	}

	if snapshot.FallbackDepth0 != 1 {
		t.Errorf("expected 1 fallback depth 0, got %d", snapshot.FallbackDepth0)
	}

	if snapshot.FallbackDepth1 != 1 {
		t.Errorf("expected 1 fallback depth 1, got %d", snapshot.FallbackDepth1)
	}

	if snapshot.RankDurationAvgUs != 1000 {
		t.Errorf("expected avg duration 1000us, got %d", snapshot.RankDurationAvgUs)
	}

	if snapshot.StatsSlowCount != 1 {
		t.Errorf("expected 1 slow query, got %d", snapshot.StatsSlowCount)
	}
}

func TestWeightNormalization(t *testing.T) {
	// 这个测试需要在前端，这里仅作占位
	t.Skip("Weight normalization is tested in frontend")
}
