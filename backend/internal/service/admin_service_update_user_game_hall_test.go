//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAdminServiceUpdateUserPersistsGameHallDisabled(t *testing.T) {
	base := &userRepoStub{user: &User{ID: 42, Email: "u@example.com"}}
	repo := &rpmUserRepoStub{userRepoStub: base}
	svc := &adminServiceImpl{userRepo: repo, redeemCodeRepo: &redeemRepoStub{}}
	disabled := true

	updated, err := svc.UpdateUser(context.Background(), 42, &UpdateUserInput{
		GameHallDisabled: &disabled,
	})

	require.NoError(t, err)
	require.True(t, updated.GameHallDisabled)
	require.NotNil(t, repo.lastUpdated)
	require.True(t, repo.lastUpdated.GameHallDisabled)
}

func TestAdminServiceUpdateUserCanExplicitlyEnableGameHall(t *testing.T) {
	base := &userRepoStub{user: &User{ID: 42, Email: "u@example.com", GameHallDisabled: true}}
	repo := &rpmUserRepoStub{userRepoStub: base}
	svc := &adminServiceImpl{userRepo: repo, redeemCodeRepo: &redeemRepoStub{}}
	disabled := false

	updated, err := svc.UpdateUser(context.Background(), 42, &UpdateUserInput{
		GameHallDisabled: &disabled,
	})

	require.NoError(t, err)
	require.False(t, updated.GameHallDisabled)
	require.NotNil(t, repo.lastUpdated)
	require.False(t, repo.lastUpdated.GameHallDisabled)
}
