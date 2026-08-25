package registrationprotection

import (
	"testing"

	registrationsettings "github.com/Wei-Shaw/sub2api/internal/custom/modules/registration-protection/settings"
)

func TestNormalizeNetworkIPv4Uses24(t *testing.T) {
	got, ok := normalizeNetwork("203.0.113.37")
	if !ok || got != "203.0.113.0/24" {
		t.Fatalf("normalizeNetwork() = %q, %v", got, ok)
	}
}

func TestNormalizeNetworkIPv6Uses64(t *testing.T) {
	got, ok := normalizeNetwork("2001:db8:abcd:1234:1:2:3:4")
	if !ok || got != "2001:db8:abcd:1234::/64" {
		t.Fatalf("normalizeNetwork() = %q, %v", got, ok)
	}
}

func TestHashValueDoesNotExposeInput(t *testing.T) {
	const input = "203.0.113.0/24"
	if got := hashValue(input); got == input || len(got) != 64 {
		t.Fatalf("hashValue() = %q", got)
	}
}

func TestScoreRiskDecisionIncludesCurrentRegistration(t *testing.T) {
	config := registrationsettings.Default()
	decision := scoreRiskDecision(config, "network", "ua", "203.0.113.0/24", 3, 5, false)
	if decision.Action != RiskActionMedium {
		t.Fatalf("cluster-only activity must remain observable instead of blocking, got %q (%d)", decision.Action, decision.Score)
	}
	if len(decision.Signals) != 2 {
		t.Fatalf("expected both cluster signals, got %#v", decision.Signals)
	}
}

func TestScoreRiskDecisionSharedNetworkDoesNotBlockWithoutExactRepeat(t *testing.T) {
	config := registrationsettings.Default()
	decision := scoreRiskDecision(config, "network", "ua", "203.0.113.0/24", 20, 40, false)
	if decision.Action == RiskActionHigh {
		t.Fatalf("shared network activity must not trigger high-risk registration rejection")
	}
}

func TestScoreRiskDecisionExactRepeatBlocksAtMediumThreshold(t *testing.T) {
	config := registrationsettings.Default()
	decision := scoreRiskDecision(config, "network", "ua", "203.0.113.0/24", 0, 0, true)
	if decision.Action != RiskActionNone {
		t.Fatalf("exact repeat without cluster history should remain non-blocking, got %q", decision.Action)
	}
	decision = scoreRiskDecision(config, "network", "ua", "203.0.113.0/24", config.ClusterWindow6h, 0, true)
	if decision.Action != RiskActionHigh {
		t.Fatalf("exact repeat at the medium cluster threshold should be high, got %q", decision.Action)
	}
}
