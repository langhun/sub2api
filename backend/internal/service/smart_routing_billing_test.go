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

type smartRoutingPermissionUserRepoStub struct {
	UserRepository
	user *User
	err  error
}

func (s *smartRoutingPermissionUserRepoStub) GetByID(context.Context, int64) (*User, error) {
	return s.user, s.err
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

func TestCheckSmartRoutingGroupPermission_StandardRevocationIsObserved(t *testing.T) {
	const userID = int64(20)
	userRepo := &smartRoutingPermissionUserRepoStub{user: &User{
		ID: userID, Status: StatusActive, RestrictPublicGroups: true,
	}}
	group := &Group{ID: 30, IsExclusive: false, SubscriptionType: SubscriptionTypeStandard}

	ok, err := CheckSmartRoutingGroupPermission(context.Background(), userRepo, nil, userID, group)
	require.NoError(t, err)
	require.False(t, ok, "a revoked public group must be rejected after the fresh user lookup")

	userRepo.user.AllowedGroups = []int64{group.ID}
	ok, err = CheckSmartRoutingGroupPermission(context.Background(), userRepo, nil, userID, group)
	require.NoError(t, err)
	require.True(t, ok)
}

func TestCheckSmartRoutingGroupPermission_SubscriptionRequiresFreshActiveSubscription(t *testing.T) {
	const userID = int64(21)
	userRepo := &smartRoutingPermissionUserRepoStub{user: &User{ID: userID, Status: StatusActive}}
	group := &Group{ID: 31, SubscriptionType: SubscriptionTypeSubscription}
	subRepo := &smartRoutingBillingSubscriptionRepoStub{}

	ok, err := CheckSmartRoutingGroupPermission(context.Background(), userRepo, subRepo, userID, group)
	require.NoError(t, err)
	require.False(t, ok)

	subRepo.sub = &UserSubscription{
		UserID: userID, GroupID: group.ID, Status: SubscriptionStatusActive,
		ExpiresAt: time.Now().Add(time.Hour),
	}
	ok, err = CheckSmartRoutingGroupPermission(context.Background(), userRepo, subRepo, userID, group)
	require.NoError(t, err)
	require.True(t, ok)
}

func TestCheckSmartRoutingGroupPermissionRepositoryFailureFailsClosed(t *testing.T) {
	repoErr := errors.New("user database unavailable")
	userRepo := &smartRoutingPermissionUserRepoStub{err: repoErr}
	ok, err := CheckSmartRoutingGroupPermission(context.Background(), userRepo, nil, 22, &Group{
		ID: 32, SubscriptionType: SubscriptionTypeStandard,
	})
	require.ErrorIs(t, err, repoErr)
	require.False(t, ok)
}
