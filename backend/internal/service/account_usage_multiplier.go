package service

import (
	"math"
	"sync"
	"time"
)

// AccountUsageMultiplierResolver is the narrow Overlay hook used when a usage
// record snapshots the upstream account cost multiplier. Core billing keeps its
// manual account multiplier as the fallback and has no knowledge of modules.
type AccountUsageMultiplierResolver interface {
	ResolveAccountUsageMultiplier(account *Account, now time.Time) (float64, bool)
}

var accountUsageMultiplierResolver struct {
	sync.RWMutex
	value AccountUsageMultiplierResolver
}

// SetAccountUsageMultiplierResolver mounts or removes the optional Overlay
// resolver at application composition time.
func SetAccountUsageMultiplierResolver(resolver AccountUsageMultiplierResolver) {
	accountUsageMultiplierResolver.Lock()
	defer accountUsageMultiplierResolver.Unlock()
	accountUsageMultiplierResolver.value = resolver
}

func resolveAccountUsageMultiplier(account *Account, now time.Time) float64 {
	fallback := account.BillingRateMultiplier()

	accountUsageMultiplierResolver.RLock()
	resolver := accountUsageMultiplierResolver.value
	accountUsageMultiplierResolver.RUnlock()
	if resolver == nil {
		return fallback
	}

	multiplier, ok := resolver.ResolveAccountUsageMultiplier(account, now)
	if !ok || multiplier < 0 || math.IsNaN(multiplier) || math.IsInf(multiplier, 0) {
		return fallback
	}
	return multiplier
}
