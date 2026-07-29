package accountdrain

import (
	"context"
	"sync"
	"time"
)

// Policy keeps the gateway hot path in memory. Repository reads happen only at
// startup and after an administrator changes a plan.
type Policy struct {
	mu      sync.RWMutex
	targets map[int64]time.Time
}

func NewPolicy() *Policy {
	return &Policy{targets: make(map[int64]time.Time)}
}

func (p *Policy) Replace(plans []Plan) {
	targets := make(map[int64]time.Time)
	for _, plan := range plans {
		if plan.Status != StatusActive {
			continue
		}
		expiresAt := time.Time{}
		if plan.ExpiresAt != nil {
			expiresAt = plan.ExpiresAt.UTC()
			if !expiresAt.After(time.Now().UTC()) {
				continue
			}
		}
		for _, accountID := range plan.AccountIDs {
			if accountID <= 0 {
				continue
			}
			previous, exists := targets[accountID]
			if !exists || previous.IsZero() || (!expiresAt.IsZero() && expiresAt.After(previous)) {
				targets[accountID] = expiresAt
			}
		}
	}
	p.mu.Lock()
	p.targets = targets
	p.mu.Unlock()
}

// PreferOpenAIAccount implements service.OpenAIAccountSchedulingPolicy.
// group and model are intentionally accepted for future additive scoping while
// current plans remain account-wide within the existing account group rules.
func (p *Policy) PreferOpenAIAccount(_ context.Context, _ *int64, _ string, accountID int64) bool {
	if p == nil || accountID <= 0 {
		return false
	}
	p.mu.RLock()
	expiresAt, ok := p.targets[accountID]
	p.mu.RUnlock()
	return ok && (expiresAt.IsZero() || expiresAt.After(time.Now().UTC()))
}
