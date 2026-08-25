package handler

import "github.com/Wei-Shaw/sub2api/internal/service"

// snapshotAPIKeyForUsage freezes the request's selected group before an
// asynchronous usage worker is queued. Smart routing mutates the request-local
// API key on failover and, for Responses WebSocket, on every turn; retaining
// that pointer in a worker would otherwise bill an earlier response to a later
// group. The service-owned User/Group data is read-only during billing, so a
// shallow value copy is sufficient while GroupID gets its own pointer.
func snapshotAPIKeyForUsage(apiKey *service.APIKey) *service.APIKey {
	if apiKey == nil {
		return nil
	}
	snapshot := *apiKey
	if apiKey.GroupID != nil {
		groupID := *apiKey.GroupID
		snapshot.GroupID = &groupID
	}
	if apiKey.Group != nil {
		group := *apiKey.Group
		snapshot.Group = &group
	}
	if apiKey.User != nil {
		user := *apiKey.User
		snapshot.User = &user
	}
	return &snapshot
}

// snapshotSubscriptionForUsage freezes the subscription selected for this
// response. Gateway failover may clear or replace the request's current
// subscription before an async worker starts.
func snapshotSubscriptionForUsage(subscription *service.UserSubscription) *service.UserSubscription {
	if subscription == nil {
		return nil
	}
	snapshot := *subscription
	if subscription.DailyWindowStart != nil {
		value := *subscription.DailyWindowStart
		snapshot.DailyWindowStart = &value
	}
	if subscription.WeeklyWindowStart != nil {
		value := *subscription.WeeklyWindowStart
		snapshot.WeeklyWindowStart = &value
	}
	if subscription.MonthlyWindowStart != nil {
		value := *subscription.MonthlyWindowStart
		snapshot.MonthlyWindowStart = &value
	}
	if subscription.AssignedBy != nil {
		value := *subscription.AssignedBy
		snapshot.AssignedBy = &value
	}
	return &snapshot
}

// snapshotAccountForUsage freezes the selected credential/account metadata
// before handing a usage task to the worker pool. WebSocket failover can
// replace the account variable while an earlier turn is still being billed.
func snapshotAccountForUsage(account *service.Account) *service.Account {
	if account == nil {
		return nil
	}
	snapshot := *account
	if account.ProxyID != nil {
		value := *account.ProxyID
		snapshot.ProxyID = &value
	}
	if account.RateMultiplier != nil {
		value := *account.RateMultiplier
		snapshot.RateMultiplier = &value
	}
	if account.LoadFactor != nil {
		value := *account.LoadFactor
		snapshot.LoadFactor = &value
	}
	return &snapshot
}
