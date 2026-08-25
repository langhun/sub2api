package smartrouting

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestValidateUpdateWeightsAndCandidates(t *testing.T) {
	valid := UpdateRequest{Mode: "smart", Strategy: "auto", PriceWeight: 40, SpeedWeight: 30, SuccessWeight: 30, StickyTTL: 1800, ExplorationRate: 0.05, GroupIDs: []int64{2}}
	if err := validateUpdate(valid); err != nil {
		t.Fatalf("valid request rejected: %v", err)
	}
	invalid := valid
	invalid.SuccessWeight = 20
	if err := validateUpdate(invalid); err == nil {
		t.Fatal("expected weight validation error")
	}
	invalid = valid
	invalid.GroupIDs = nil
	if err := validateUpdate(invalid); err == nil {
		t.Fatal("expected candidate validation error")
	}
}

func TestValidateUpdateRejectsExplorationAboveCap(t *testing.T) {
	req := UpdateRequest{Mode: "single", Strategy: "auto", PriceWeight: 40, SpeedWeight: 30, SuccessWeight: 30, ExplorationRate: 0.06}
	if err := validateUpdate(req); err == nil {
		t.Fatal("expected exploration cap validation error")
	}
}

func TestRouteStateHardLimitsSwitchesAndContinuity(t *testing.T) {
	state := &service.SmartRoutingRequestState{MainGroupID: 1, CurrentGroupID: 2, GroupSwitches: 2, ContinuityLocked: false}
	if routeStateAllowsGroup(state, 3, 0, 2) {
		t.Fatal("expected third group switch to be blocked")
	}
	state.GroupSwitches = 1
	if !routeStateAllowsGroup(state, 3, 0, 2) {
		t.Fatal("expected second group switch to remain allowed")
	}
	state.ContinuityLocked = true
	if routeStateAllowsGroup(state, 3, 2, 2) {
		t.Fatal("continuation must stay on sticky group")
	}
}
