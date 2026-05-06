package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildProxyAssignmentPlanBalancesSelectedProxies(t *testing.T) {
	proxies := []Proxy{
		{ID: 1, Name: "p1"},
		{ID: 2, Name: "p2"},
		{ID: 3, Name: "p3"},
	}
	counts := map[int64]int64{1: 1, 2: 3, 3: 2}
	targets := []proxyAssignmentTargetAccount{
		{ID: 101, Name: "a1"},
		{ID: 102, Name: "a2"},
		{ID: 103, Name: "a3"},
		{ID: 104, Name: "a4"},
		{ID: 105, Name: "a5"},
		{ID: 106, Name: "a6"},
	}

	result := buildProxyAssignmentPlan(true, proxies, counts, targets)

	require.Equal(t, 6, result.PlannedAssignmentCount)
	finalCounts := []int64{
		result.Proxies[0].AfterAccountCount,
		result.Proxies[1].AfterAccountCount,
		result.Proxies[2].AfterAccountCount,
	}
	require.LessOrEqual(t, assignmentRange(finalCounts), int64(1))
	require.Equal(t, int64(4), result.Proxies[0].AfterAccountCount)
	require.Equal(t, int64(4), result.Proxies[1].AfterAccountCount)
	require.Equal(t, int64(4), result.Proxies[2].AfterAccountCount)
}

func TestMarkAppliedProxyAssignmentsKeepsSkippedAccountsVisible(t *testing.T) {
	result := buildProxyAssignmentPlan(false, []Proxy{{ID: 1, Name: "p1"}}, map[int64]int64{1: 0}, []proxyAssignmentTargetAccount{
		{ID: 101, Name: "a1"},
		{ID: 102, Name: "a2"},
	})

	markAppliedProxyAssignments(result, map[int64]bool{101: true})

	require.Equal(t, 1, result.ActualAssignmentCount)
	require.Equal(t, 1, result.Proxies[0].AssignedCount)
	require.Equal(t, int64(1), result.Proxies[0].AfterAccountCount)
	require.True(t, result.Proxies[0].Accounts[0].Assigned)
	require.False(t, result.Proxies[0].Accounts[1].Assigned)
	require.NotEmpty(t, result.Proxies[0].Accounts[1].SkippedReason)
}

func TestDedupePositiveInt64sDropsDuplicateAndInvalidIDs(t *testing.T) {
	got := dedupePositiveInt64s([]int64{3, 0, 2, 3, -1, 2, 4})
	require.Equal(t, []int64{3, 2, 4}, got)
}
