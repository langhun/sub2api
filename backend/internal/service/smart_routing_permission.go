package service

import (
	"context"
	"errors"
	"fmt"
)

// ErrSmartRoutingPermissionUnavailable is returned when a candidate cannot be
// re-authorized against the current user/subscription state. A routed request
// must fail closed in this case; using the stale authentication snapshot would
// turn a permission revocation into an active billing bypass.
var ErrSmartRoutingPermissionUnavailable = errors.New("smart routing permission check unavailable")

// CheckSmartRoutingGroupPermission performs the final, request-time candidate
// authorization check. The API-key authentication snapshot is intentionally
// not used here because user allowed-groups and subscriptions may be revoked
// after authentication but before the Overlay candidate is dispatched.
//
// It returns (false, nil) for a valid repository lookup that proves the user
// no longer has access, and a non-nil error when the authorization state could
// not be established. Callers must reject both cases.
func CheckSmartRoutingGroupPermission(
	ctx context.Context,
	userRepo UserRepository,
	userSubscriptionRepo UserSubscriptionRepository,
	userID int64,
	group *Group,
) (bool, error) {
	if userRepo == nil || group == nil || userID <= 0 {
		return false, ErrSmartRoutingPermissionUnavailable
	}
	if !smartRoutingKnownSubscriptionType(group.SubscriptionType) {
		return false, nil
	}

	user, err := userRepo.GetByID(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("load smart routing user permission: %w", err)
	}
	if user == nil || !user.IsActive() {
		return false, nil
	}

	if group.SubscriptionType == SubscriptionTypeSubscription {
		if userSubscriptionRepo == nil {
			return false, ErrSmartRoutingPermissionUnavailable
		}
		subscription, err := userSubscriptionRepo.GetActiveByUserIDAndGroupID(ctx, userID, group.ID)
		if err != nil {
			if errors.Is(err, ErrSubscriptionNotFound) {
				return false, nil
			}
			return false, fmt.Errorf("load smart routing subscription: %w", err)
		}
		return subscription != nil && subscription.IsActive(), nil
	}

	return user.CanBindGroup(group.ID, group.IsExclusive), nil
}
