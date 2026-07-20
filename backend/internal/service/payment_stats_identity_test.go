package service

import (
	"testing"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/stretchr/testify/require"
)

func TestBuildTopUsersIncludesUsernameSnapshot(t *testing.T) {
	orders := []*dbent.PaymentOrder{
		{UserID: 7, UserEmail: "alice@example.com", PayAmount: 10.25},
		{UserID: 7, UserEmail: "alice@example.com", UserName: "alice", PayAmount: 2.25},
		{UserID: 8, UserEmail: "fallback@example.com", PayAmount: 5},
	}

	users := buildTopUsers(orders)
	require.Len(t, users, 2)
	require.Equal(t, TopUserStat{
		UserID: 7, Email: "alice@example.com", Username: "alice", Amount: 12.5,
	}, users[0])
	require.Equal(t, "", users[1].Username)
	require.Equal(t, "fallback@example.com", users[1].Email)
}
