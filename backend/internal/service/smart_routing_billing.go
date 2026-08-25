package service

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

// CheckSmartRoutingGroupEligibility validates the billing boundary for a
// candidate selected after API-key authentication. The middleware already
// checked the primary group (including key/user-wide limits), so this method
// deliberately checks only group-specific balance/subscription state and the
// candidate's own RPM bucket. Re-running the full preflight here would count
// the same request multiple times against API-key and user-wide limits.
func (s *BillingCacheService) CheckSmartRoutingGroupEligibility(
	ctx context.Context,
	user *User,
	apiKey *APIKey,
	group *Group,
) error {
	if s == nil {
		return ErrBillingServiceUnavailable
	}
	if s.cfg == nil {
		return ErrBillingServiceUnavailable
	}
	if s.cfg.RunMode == config.RunModeSimple {
		return nil
	}
	if user == nil || group == nil || user.ID <= 0 || group.ID <= 0 {
		return ErrBillingServiceUnavailable
	}
	if s.circuitBreaker != nil && !s.circuitBreaker.Allow() {
		return ErrBillingServiceUnavailable
	}

	if group.IsSubscriptionType() {
		if s.subRepo == nil {
			return ErrBillingServiceUnavailable
		}
		status, err := s.GetSubscriptionStatus(ctx, user.ID, group.ID)
		if err != nil {
			return ErrBillingServiceUnavailable.WithCause(err)
		}
		if status == nil || status.Status != SubscriptionStatusActive || time.Now().After(status.ExpiresAt) {
			return ErrSubscriptionInvalid
		}
		if group.HasDailyLimit() && status.DailyUsage >= *group.DailyLimitUSD {
			return ErrDailyLimitExceeded
		}
		if group.HasWeeklyLimit() && status.WeeklyUsage >= *group.WeeklyLimitUSD {
			return ErrWeeklyLimitExceeded
		}
		if group.HasMonthlyLimit() && status.MonthlyUsage >= *group.MonthlyLimitUSD {
			return ErrMonthlyLimitExceeded
		}
	} else {
		if err := s.checkBalanceEligibility(ctx, user.ID); err != nil {
			return err
		}
	}

	// Group RPM is the only request counter that changes with the selected
	// candidate. API-key and user-wide RPM were already charged by the primary
	// preflight and must not be incremented again here.
	if err := s.checkGroupRPMOnly(ctx, user, group); err != nil {
		return err
	}
	_ = apiKey // kept in the signature to make the billing boundary explicit
	return nil
}

func (s *BillingCacheService) checkGroupRPMOnly(ctx context.Context, user *User, group *Group) error {
	if s == nil || user == nil || group == nil || s.userRPMCache == nil {
		return nil
	}
	var override *int
	// The auth snapshot's override belongs to the primary group. A candidate
	// may have a different user×group override, so never reuse that snapshot
	// here; resolve the selected group's value explicitly.
	if s.userGroupRateRepo != nil {
		value, err := s.userGroupRateRepo.GetRPMOverrideByUserAndGroup(ctx, user.ID, group.ID)
		if err == nil {
			override = value
		}
	}
	if override != nil {
		if *override <= 0 {
			return nil
		}
		count, err := s.userRPMCache.IncrementUserGroupRPM(ctx, user.ID, group.ID)
		if err != nil {
			return nil // preserve the existing group-RPM fail-open policy
		}
		if count > *override {
			return ErrGroupRPMExceeded
		}
		return nil
	}
	if group.RPMLimit <= 0 {
		return nil
	}
	count, err := s.userRPMCache.IncrementUserGroupRPM(ctx, user.ID, group.ID)
	if err != nil {
		return nil
	}
	if count > group.RPMLimit {
		return ErrGroupRPMExceeded
	}
	return nil
}
