package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type smartRoutingBillingSubscriptionRepoStub struct {
	UserSubscriptionRepository
	sub *UserSubscription
	err error
}

func (s *smartRoutingBillingSubscriptionRepoStub) GetActiveByUserIDAndGroupID(context.Context, int64, int64) (*UserSubscription, error) {
	return s.sub, s.err
}

func newSmartRoutingBillingService(repo UserSubscriptionRepository) *BillingCacheService {
	return &BillingCacheService{
		subRepo: repo,
		cfg:     &config.Config{RunMode: config.RunModeStandard},
	}
}

func TestCheckSmartRoutingGroupEligibility_SubscriptionWithinLimit(t *testing.T) {
	limit := 10.0
	svc := newSmartRoutingBillingService(&smartRoutingBillingSubscriptionRepoStub{
		sub: &UserSubscription{
			ID:              10,
			UserID:          20,
			GroupID:         30,
			Status:          SubscriptionStatusActive,
			ExpiresAt:       time.Now().Add(time.Hour),
			DailyUsageUSD:   2,
			WeeklyUsageUSD:  3,
			MonthlyUsageUSD: 4,
		},
	})

	err := svc.CheckSmartRoutingGroupEligibility(context.Background(), &User{ID: 20}, &APIKey{ID: 40}, &Group{
		ID:               30,
		SubscriptionType: SubscriptionTypeSubscription,
		DailyLimitUSD:    &limit,
	})
	require.NoError(t, err)
}

func TestCheckSmartRoutingGroupEligibility_SubscriptionLimitExceeded(t *testing.T) {
	limit := 10.0
	svc := newSmartRoutingBillingService(&smartRoutingBillingSubscriptionRepoStub{
		sub: &UserSubscription{
			ID:            10,
			UserID:        20,
			GroupID:       30,
			Status:        SubscriptionStatusActive,
			ExpiresAt:     time.Now().Add(time.Hour),
			DailyUsageUSD: limit,
		},
	})

	err := svc.CheckSmartRoutingGroupEligibility(context.Background(), &User{ID: 20}, &APIKey{ID: 40}, &Group{
		ID:               30,
		SubscriptionType: SubscriptionTypeSubscription,
		DailyLimitUSD:    &limit,
	})
	require.ErrorIs(t, err, ErrDailyLimitExceeded)
}

func TestCheckSmartRoutingGroupEligibility_MissingSubscriptionFailsClosed(t *testing.T) {
	svc := newSmartRoutingBillingService(&smartRoutingBillingSubscriptionRepoStub{})

	err := svc.CheckSmartRoutingGroupEligibility(context.Background(), &User{ID: 20}, &APIKey{ID: 40}, &Group{
		ID:               30,
		SubscriptionType: SubscriptionTypeSubscription,
	})
	require.ErrorIs(t, err, ErrSubscriptionInvalid)
}

func TestCheckSmartRoutingGroupEligibility_SubscriptionRepositoryErrorFailsClosed(t *testing.T) {
	svc := newSmartRoutingBillingService(&smartRoutingBillingSubscriptionRepoStub{err: errors.New("database unavailable")})

	err := svc.CheckSmartRoutingGroupEligibility(context.Background(), &User{ID: 20}, &APIKey{ID: 40}, &Group{
		ID:               30,
		SubscriptionType: SubscriptionTypeSubscription,
	})
	require.ErrorContains(t, err, "BILLING_SERVICE_ERROR")
}
