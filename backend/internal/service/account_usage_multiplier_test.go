package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type accountUsageMultiplierResolverStub struct {
	multiplier float64
	ok         bool
}

func (s accountUsageMultiplierResolverStub) ResolveAccountUsageMultiplier(*Account, time.Time) (float64, bool) {
	return s.multiplier, s.ok
}

func TestResolveAccountUsageMultiplierUsesResolverAndFallsBack(t *testing.T) {
	t.Cleanup(func() { SetAccountUsageMultiplierResolver(nil) })
	manual := 1.0
	account := &Account{RateMultiplier: &manual}
	now := time.Now()

	SetAccountUsageMultiplierResolver(accountUsageMultiplierResolverStub{multiplier: 0.08, ok: true})
	require.Equal(t, 0.08, resolveAccountUsageMultiplier(account, now))

	SetAccountUsageMultiplierResolver(accountUsageMultiplierResolverStub{ok: false})
	require.Equal(t, 1.0, resolveAccountUsageMultiplier(account, now))
}
