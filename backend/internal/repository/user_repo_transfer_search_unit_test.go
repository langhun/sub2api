package repository

import (
	"context"
	"fmt"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func createTransferSearchUser(t *testing.T, repo *userRepository, username, email, status string) *service.User {
	t.Helper()
	user := &service.User{
		Email:        email,
		Username:     username,
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       status,
	}
	require.NoError(t, repo.Create(context.Background(), user))
	return user
}

func TestUserRepositorySearchActiveTransferReceiversMatchesFragmentsAndExcludesUnsafeRows(t *testing.T) {
	repo, _ := newUserEntRepo(t)
	requester := createTransferSearchUser(t, repo, "alice-self", "self@example.com", service.StatusActive)
	usernameMatch := createTransferSearchUser(t, repo, "AliceWonder", "wonder@example.com", service.StatusActive)
	emailMatch := createTransferSearchUser(t, repo, "other", "contact+alice@example.com", service.StatusActive)
	createTransferSearchUser(t, repo, "alice-disabled", "disabled@example.com", service.StatusDisabled)
	createTransferSearchUser(t, repo, "unrelated", "none@example.com", service.StatusActive)

	got, err := repo.SearchActiveTransferReceivers(context.Background(), "ALICE", requester.ID, 8)

	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, []int64{usernameMatch.ID, emailMatch.ID}, []int64{got[0].ID, got[1].ID})
}

func TestUserRepositorySearchActiveTransferReceiversEnforcesLimit(t *testing.T) {
	repo, _ := newUserEntRepo(t)
	requester := createTransferSearchUser(t, repo, "requester", "requester@example.com", service.StatusActive)
	for i := 0; i < 12; i++ {
		createTransferSearchUser(t, repo, fmt.Sprintf("match-%02d", i), fmt.Sprintf("match-%02d@example.com", i), service.StatusActive)
	}

	got, err := repo.SearchActiveTransferReceivers(context.Background(), "match", requester.ID, 8)

	require.NoError(t, err)
	require.Len(t, got, 8)
}

func TestUserRepositorySearchActiveTransferReceiversMatchesNumericID(t *testing.T) {
	repo, _ := newUserEntRepo(t)
	requester := createTransferSearchUser(t, repo, "requester", "requester@example.com", service.StatusActive)
	target := createTransferSearchUser(t, repo, "target", "target@example.com", service.StatusActive)

	got, err := repo.SearchActiveTransferReceivers(context.Background(), fmt.Sprint(target.ID), requester.ID, 8)

	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, target.ID, got[0].ID)
}
