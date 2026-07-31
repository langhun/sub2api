// Package upstreamcost owns the downstream accounting adapter for upstream
// billing probes.
package upstreamcost

import (
	"context"
	"fmt"
	"math"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// RateMultiplierWriter is the only core write capability used by this module.
type RateMultiplierWriter interface {
	BulkUpdate(context.Context, []int64, service.AccountBulkUpdate) (int64, error)
}

// RateSyncer applies a successfully discovered per-Key multiplier to the
// existing account setting. All core accounting paths then use the same field.
type RateSyncer struct {
	accounts RateMultiplierWriter
}

func NewRateSyncer(accounts RateMultiplierWriter) *RateSyncer {
	return &RateSyncer{accounts: accounts}
}

func (s *RateSyncer) OnUpstreamBillingProbeSuccess(ctx context.Context, account *service.Account, snapshot *service.UpstreamBillingProbeSnapshot) error {
	if s == nil || s.accounts == nil || account == nil || snapshot == nil || snapshot.Status != service.UpstreamBillingProbeStatusOK {
		return nil
	}
	multiplier, ok := snapshotMultiplier(snapshot)
	if !ok {
		return nil
	}

	updated, err := s.accounts.BulkUpdate(ctx, []int64{account.ID}, service.AccountBulkUpdate{
		RateMultiplier: &multiplier,
	})
	if err != nil {
		return fmt.Errorf("sync account rate multiplier: %w", err)
	}
	if updated != 1 {
		return fmt.Errorf("sync account rate multiplier: account %d was not updated", account.ID)
	}
	return nil
}

func snapshotMultiplier(snapshot *service.UpstreamBillingProbeSnapshot) (float64, bool) {
	if snapshot == nil || snapshot.Data == nil || snapshot.Data["billing_scope"] != "token" {
		return 0, false
	}
	multiplier, ok := snapshot.Data["effective_rate_multiplier"].(float64)
	if !ok || multiplier < 0 || math.IsNaN(multiplier) || math.IsInf(multiplier, 0) {
		return 0, false
	}
	return multiplier, true
}
