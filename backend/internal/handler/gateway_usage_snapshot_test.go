package handler

import (
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestSnapshotAPIKeyForUsageFreezesSelectedGroup(t *testing.T) {
	selectedID := int64(11)
	key := &service.APIKey{
		ID:      7,
		GroupID: &selectedID,
		Group:   &service.Group{ID: selectedID, Name: "selected"},
		User:    &service.User{ID: 3},
	}

	snapshot := snapshotAPIKeyForUsage(key)
	require.NotSame(t, key, snapshot)
	require.NotSame(t, key.GroupID, snapshot.GroupID)
	require.NotSame(t, key.Group, snapshot.Group)
	require.NotSame(t, key.User, snapshot.User)

	newID := int64(12)
	key.GroupID = &newID
	key.Group = &service.Group{ID: newID, Name: "later"}
	key.User.ID = 99

	require.Equal(t, selectedID, *snapshot.GroupID)
	require.Equal(t, selectedID, snapshot.Group.ID)
	require.Equal(t, "selected", snapshot.Group.Name)
	require.Equal(t, int64(3), snapshot.User.ID)
}

func TestUsageSnapshotsFreezeMutableRoutingState(t *testing.T) {
	groupID := int64(21)
	originalGroupID := groupID
	accountID := int64(31)
	subscriptionID := int64(41)
	window := time.Now()
	originalWindow := window
	account := &service.Account{ID: accountID, RateMultiplier: ptrFloat64(1.25), ProxyID: &groupID}
	subscription := &service.UserSubscription{ID: subscriptionID, GroupID: groupID, DailyWindowStart: &window}

	accountSnapshot := snapshotAccountForUsage(account)
	subscriptionSnapshot := snapshotSubscriptionForUsage(subscription)
	*account.RateMultiplier = 3
	*account.ProxyID = 99
	*subscription.DailyWindowStart = window.Add(24 * time.Hour)

	require.Equal(t, accountID, accountSnapshot.ID)
	require.Equal(t, 1.25, *accountSnapshot.RateMultiplier)
	require.Equal(t, originalGroupID, *accountSnapshot.ProxyID)
	require.Equal(t, subscriptionID, subscriptionSnapshot.ID)
	require.Equal(t, originalWindow, *subscriptionSnapshot.DailyWindowStart)
}

func ptrFloat64(value float64) *float64 { return &value }
