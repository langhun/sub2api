package registrationprotection

import (
	"context"
	"testing"

	registrationsettings "github.com/Wei-Shaw/sub2api/internal/custom/modules/registration-protection/settings"
	"github.com/Wei-Shaw/sub2api/internal/custom/settings/contract"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type policySettingsStub struct{ values map[string]string }

func (s policySettingsStub) GetMultiple(context.Context, []string) (map[string]string, error) {
	return s.values, nil
}
func (s policySettingsStub) SetMultiple(context.Context, map[string]string) error { return nil }

var _ contract.Store = policySettingsStub{}

func TestPolicyRejectsCommonDisposableEmailDomains(t *testing.T) {
	policy := NewPolicy(policySettingsStub{values: map[string]string{}})
	if err := policy.Validate(context.Background(), "user@mailinator.com"); err != service.ErrEmailDomainRegistrationLimit {
		t.Fatalf("expected disposable domain rejection, got %v", err)
	}
}

func TestPolicyKeepsOrdinaryEnterpriseEmailAllowedWhenAliasRuleDisabled(t *testing.T) {
	policy := NewPolicy(policySettingsStub{values: map[string]string{
		registrationsettings.KeyAliasEnabled: "false",
	}})
	if err := policy.Validate(context.Background(), "user@enterprise.example"); err != nil {
		t.Fatalf("ordinary enterprise email should remain allowed, got %v", err)
	}
}

func TestRiskReservationBlocksRewardsForMediumDecision(t *testing.T) {
	reservation := &riskReservation{decision: RiskDecision{Action: RiskActionMedium}}
	if !reservation.RewardBlocked() {
		t.Fatal("medium-risk registrations must not receive signup rewards")
	}
}
