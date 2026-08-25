package smartrouting

import "testing"

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
