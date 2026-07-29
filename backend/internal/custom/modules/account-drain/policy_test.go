package accountdrain

import (
	"context"
	"testing"
	"time"
)

func TestPolicyPrefersOnlyActiveAndUnexpiredTargets(t *testing.T) {
	policy := NewPolicy()
	future := time.Now().Add(time.Hour)
	past := time.Now().Add(-time.Hour)
	policy.Replace([]Plan{
		{Status: StatusActive, AccountIDs: []int64{10}},
		{Status: StatusStopped, AccountIDs: []int64{11}},
		{Status: StatusActive, AccountIDs: []int64{12}, ExpiresAt: &past},
		{Status: StatusActive, AccountIDs: []int64{13}, ExpiresAt: &future},
	})

	for accountID, want := range map[int64]bool{10: true, 11: false, 12: false, 13: true, 14: false} {
		if got := policy.PreferOpenAIAccount(context.Background(), nil, "gpt-5", accountID); got != want {
			t.Errorf("PreferOpenAIAccount(%d) = %v, want %v", accountID, got, want)
		}
	}
}
