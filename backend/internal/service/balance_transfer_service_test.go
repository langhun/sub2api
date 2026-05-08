package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type transferSettingRepoStub struct {
	settings map[string]string
}

func (s *transferSettingRepoStub) Get(context.Context, string) (*Setting, error) { return nil, nil }

func (s *transferSettingRepoStub) GetValue(context.Context, string) (string, error) {
	return "", nil
}

func (s *transferSettingRepoStub) Set(context.Context, string, string) error { return nil }

func (s *transferSettingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	values := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.settings[key]; ok {
			values[key] = value
		}
	}
	return values, nil
}

func (s *transferSettingRepoStub) SetMultiple(context.Context, map[string]string) error { return nil }

func (s *transferSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	values := make(map[string]string, len(s.settings))
	for key, value := range s.settings {
		values[key] = value
	}
	return values, nil
}

func (s *transferSettingRepoStub) Delete(context.Context, string) error { return nil }

type transferBalanceMutation struct {
	userID int64
	amount float64
}

type transferUserRepoStub struct {
	users       map[int64]*User
	deductCalls []transferBalanceMutation
	updateCalls []transferBalanceMutation
}

func (s *transferUserRepoStub) Create(context.Context, *User) error { return nil }

func (s *transferUserRepoStub) GetByID(_ context.Context, id int64) (*User, error) {
	user, ok := s.users[id]
	if !ok {
		return nil, ErrUserNotFound
	}
	cloned := *user
	return &cloned, nil
}

func (s *transferUserRepoStub) GetByEmail(context.Context, string) (*User, error) {
	return nil, ErrUserNotFound
}

func (s *transferUserRepoStub) GetFirstAdmin(context.Context) (*User, error) {
	return nil, ErrUserNotFound
}

func (s *transferUserRepoStub) Update(context.Context, *User) error { return nil }

func (s *transferUserRepoStub) Delete(context.Context, int64) error { return nil }

func (s *transferUserRepoStub) GetUserAvatar(context.Context, int64) (*UserAvatar, error) {
	return nil, nil
}

func (s *transferUserRepoStub) UpsertUserAvatar(context.Context, int64, UpsertUserAvatarInput) (*UserAvatar, error) {
	return nil, nil
}

func (s *transferUserRepoStub) DeleteUserAvatar(context.Context, int64) error { return nil }

func (s *transferUserRepoStub) List(context.Context, pagination.PaginationParams) ([]User, *pagination.PaginationResult, error) {
	return nil, nil, nil
}

func (s *transferUserRepoStub) ListWithFilters(context.Context, pagination.PaginationParams, UserListFilters) ([]User, *pagination.PaginationResult, error) {
	return nil, nil, nil
}

func (s *transferUserRepoStub) GetLatestUsedAtByUserIDs(context.Context, []int64) (map[int64]*time.Time, error) {
	return map[int64]*time.Time{}, nil
}

func (s *transferUserRepoStub) GetLatestUsedAtByUserID(context.Context, int64) (*time.Time, error) {
	return nil, nil
}

func (s *transferUserRepoStub) UpdateUserLastActiveAt(context.Context, int64, time.Time) error {
	return nil
}

func (s *transferUserRepoStub) UpdateBalance(_ context.Context, id int64, amount float64) error {
	user, ok := s.users[id]
	if !ok {
		return ErrUserNotFound
	}
	s.updateCalls = append(s.updateCalls, transferBalanceMutation{userID: id, amount: amount})
	user.Balance += amount
	return nil
}

func (s *transferUserRepoStub) DeductBalance(_ context.Context, id int64, amount float64) error {
	user, ok := s.users[id]
	if !ok {
		return ErrUserNotFound
	}
	s.deductCalls = append(s.deductCalls, transferBalanceMutation{userID: id, amount: amount})
	user.Balance -= amount
	return nil
}

func (s *transferUserRepoStub) UpdateConcurrency(context.Context, int64, int) error { return nil }

func (s *transferUserRepoStub) BatchSetConcurrency(context.Context, []int64, int) (int, error) {
	return 0, nil
}

func (s *transferUserRepoStub) BatchAddConcurrency(context.Context, []int64, int) (int, error) {
	return 0, nil
}

func (s *transferUserRepoStub) ExistsByEmail(context.Context, string) (bool, error) {
	return false, nil
}

func (s *transferUserRepoStub) RemoveGroupFromAllowedGroups(context.Context, int64) (int64, error) {
	return 0, nil
}

func (s *transferUserRepoStub) AddGroupToAllowedGroups(context.Context, int64, int64) error {
	return nil
}

func (s *transferUserRepoStub) RemoveGroupFromUserAllowedGroups(context.Context, int64, int64) error {
	return nil
}

func (s *transferUserRepoStub) ListUserAuthIdentities(context.Context, int64) ([]UserAuthIdentityRecord, error) {
	return nil, nil
}

func (s *transferUserRepoStub) UnbindUserAuthProvider(context.Context, int64, string) error {
	return nil
}

func (s *transferUserRepoStub) UpdateTotpSecret(context.Context, int64, *string) error { return nil }

func (s *transferUserRepoStub) EnableTotp(context.Context, int64) error { return nil }

func (s *transferUserRepoStub) DisableTotp(context.Context, int64) error { return nil }

type transferRepoStub struct {
	lockedBalances      map[int64]float64
	dailyTotal          float64
	dailyCount          int
	transferForUpdate   *BalanceTransferRecord
	baseTransfer        *BalanceTransferRecord
	runInTxCalls        int
	createCalled        bool
	updateStatusCalled  bool
	updateStatusHistory []string
}

func (s *transferRepoStub) Create(_ context.Context, record *BalanceTransferRecord) error {
	s.createCalled = true
	record.ID = 1
	return nil
}

func (s *transferRepoStub) GetByID(context.Context, int64) (*BalanceTransferRecord, error) {
	if s.baseTransfer == nil {
		return nil, ErrTransferNotFound
	}
	record := *s.baseTransfer
	return &record, nil
}

func (s *transferRepoStub) GetByIDForUpdate(context.Context, int64) (*BalanceTransferRecord, error) {
	if s.transferForUpdate == nil {
		return nil, ErrTransferNotFound
	}
	record := *s.transferForUpdate
	return &record, nil
}

func (s *transferRepoStub) LockUserBalance(_ context.Context, userID int64) (float64, error) {
	balance, ok := s.lockedBalances[userID]
	if !ok {
		return 0, ErrUserNotFound
	}
	return balance, nil
}

func (s *transferRepoStub) UpdateStatus(_ context.Context, _ int64, status string, frozenAt *time.Time, frozenBy *int64, revokeReason *string) error {
	s.updateStatusCalled = true
	s.updateStatusHistory = append(s.updateStatusHistory, status)
	if s.transferForUpdate != nil {
		s.transferForUpdate.Status = status
		s.transferForUpdate.FrozenAt = frozenAt
		s.transferForUpdate.FrozenBy = frozenBy
		s.transferForUpdate.RevokeReason = revokeReason
	}
	return nil
}

func (s *transferRepoStub) ListByUser(context.Context, int64, string, int, int) ([]*BalanceTransferRecord, int, error) {
	return nil, 0, nil
}

func (s *transferRepoStub) ListByUserExcludeType(context.Context, int64, string, string, int, int) ([]*BalanceTransferRecord, int, error) {
	return nil, 0, nil
}

func (s *transferRepoStub) ListAll(context.Context, *TransferFilter, int, int) ([]*BalanceTransferRecord, int, error) {
	return nil, 0, nil
}

func (s *transferRepoStub) GetDailyTransferTotal(context.Context, int64) (float64, int, error) {
	return s.dailyTotal, s.dailyCount, nil
}

func (s *transferRepoStub) GetFeeStats(context.Context, time.Time, time.Time) ([]*DailyFeeStat, error) {
	return nil, nil
}

func (s *transferRepoStub) GetLeaderboard(context.Context, time.Time, time.Time, int, string) ([]*TransferRankEntry, error) {
	return nil, nil
}

func (s *transferRepoStub) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	s.runInTxCalls++
	return fn(ctx)
}

func (s *transferRepoStub) GetUserTransferStats(context.Context, int64) (float64, float64, float64, error) {
	return 0, 0, 0, nil
}

func newTransferTestService(
	t *testing.T,
	repo *transferRepoStub,
	userRepo *transferUserRepoStub,
	overrides map[string]string,
) *BalanceTransferService {
	t.Helper()

	settings := map[string]string{
		SettingKeyTransferEnabled:         "true",
		SettingKeyTransferFeeRate:         "0",
		SettingKeyTransferMinAmount:       "1",
		SettingKeyTransferMaxAmount:       "1000",
		SettingKeyTransferDailyLimit:      "100",
		SettingKeyTransferDailyCountLimit: "10",
		SettingKeyTransferVIPFeeExempt:    "false",
		SettingKeyRedPacketEnabled:        "false",
	}
	for key, value := range overrides {
		settings[key] = value
	}

	settingSvc := NewSettingService(&transferSettingRepoStub{settings: settings}, &config.Config{})
	return NewBalanceTransferService(repo, nil, userRepo, settingSvc)
}

func TestTransfer_RechecksDailyLimitInsideTx(t *testing.T) {
	repo := &transferRepoStub{
		lockedBalances: map[int64]float64{1: 500},
		dailyTotal:     80,
	}
	userRepo := &transferUserRepoStub{
		users: map[int64]*User{
			1: {ID: 1, Balance: 500},
			2: {ID: 2, Balance: 0},
		},
	}
	svc := newTransferTestService(t, repo, userRepo, map[string]string{
		SettingKeyTransferDailyLimit: "100",
	})

	record, err := svc.Transfer(context.Background(), 1, 2, 30, nil)
	require.Nil(t, record)
	require.ErrorIs(t, err, ErrTransferDailyLimit)
	require.Equal(t, 1, repo.runInTxCalls)
	require.False(t, repo.createCalled)
	require.Empty(t, userRepo.deductCalls)
	require.Empty(t, userRepo.updateCalls)
}

func TestTransfer_RechecksSenderBalanceInsideTx(t *testing.T) {
	repo := &transferRepoStub{
		lockedBalances: map[int64]float64{1: 20},
	}
	userRepo := &transferUserRepoStub{
		users: map[int64]*User{
			1: {ID: 1, Balance: 500},
			2: {ID: 2, Balance: 0},
		},
	}
	svc := newTransferTestService(t, repo, userRepo, nil)

	record, err := svc.Transfer(context.Background(), 1, 2, 30, nil)
	require.Nil(t, record)
	require.ErrorIs(t, err, ErrTransferInsufficient)
	require.Equal(t, 1, repo.runInTxCalls)
	require.False(t, repo.createCalled)
	require.Empty(t, userRepo.deductCalls)
	require.Empty(t, userRepo.updateCalls)
}

func TestRevokeTransfer_RechecksLockedStatusBeforeRefund(t *testing.T) {
	repo := &transferRepoStub{
		baseTransfer: &BalanceTransferRecord{
			ID:           7,
			SenderID:     1,
			ReceiverID:   2,
			Amount:       25,
			GrossAmount:  25,
			TransferType: "direct",
			Status:       "completed",
			CreatedAt:    time.Now(),
		},
		transferForUpdate: &BalanceTransferRecord{
			ID:           7,
			SenderID:     1,
			ReceiverID:   2,
			Amount:       25,
			GrossAmount:  25,
			TransferType: "direct",
			Status:       "revoked",
			CreatedAt:    time.Now(),
		},
	}
	userRepo := &transferUserRepoStub{
		users: map[int64]*User{
			1: {ID: 1, Balance: 10},
			2: {ID: 2, Balance: 100},
		},
	}
	svc := newTransferTestService(t, repo, userRepo, nil)

	err := svc.RevokeTransfer(context.Background(), 99, 7, "duplicate revoke")
	require.ErrorIs(t, err, ErrTransferAlreadyRevoked)
	require.Equal(t, 1, repo.runInTxCalls)
	require.False(t, repo.updateStatusCalled)
	require.Empty(t, userRepo.deductCalls)
	require.Empty(t, userRepo.updateCalls)
}
