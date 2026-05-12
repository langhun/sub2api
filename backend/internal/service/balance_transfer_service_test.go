package service

import (
	"context"
	"errors"
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
	users            map[int64]*User
	listUsers        []User
	listFilters      []UserListFilters
	batchEmailIDs    []int64
	deductCalls      []transferBalanceMutation
	updateCalls      []transferBalanceMutation
	getByIDCallCount int
}

func (s *transferUserRepoStub) Create(context.Context, *User) error { return nil }

func (s *transferUserRepoStub) GetByID(_ context.Context, id int64) (*User, error) {
	s.getByIDCallCount++
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

func (s *transferUserRepoStub) ListWithFilters(_ context.Context, params pagination.PaginationParams, filters UserListFilters) ([]User, *pagination.PaginationResult, error) {
	s.listFilters = append(s.listFilters, filters)
	limit := params.Limit()
	if limit <= 0 || limit > len(s.listUsers) {
		limit = len(s.listUsers)
	}
	users := append([]User(nil), s.listUsers[:limit]...)
	return users, &pagination.PaginationResult{Total: int64(len(s.listUsers)), Page: params.Page, PageSize: params.PageSize}, nil
}

func (s *transferUserRepoStub) GetEmailsByIDs(_ context.Context, userIDs []int64) (map[int64]string, error) {
	s.batchEmailIDs = append([]int64(nil), userIDs...)
	emails := make(map[int64]string, len(userIDs))
	for _, id := range userIDs {
		if user, ok := s.users[id]; ok {
			emails[id] = user.Email
		}
	}
	return emails, nil
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
	listAllRecords      []*BalanceTransferRecord
	listAllTotal        int
	runInTxCalls        int
	createCalled        bool
	updateStatusCalled  bool
	updateStatusHistory []string
}

type redPacketRepoStub struct {
	redPackets      map[int64]*RedPacketRecord
	claims          map[int64][]*RedPacketClaimRecord
	activeExpired   []*RedPacketRecord
	created         bool
	returnRemaining map[int64]float64
	returnErr       error
}

func (s *redPacketRepoStub) Create(_ context.Context, rp *RedPacketRecord) error {
	s.created = true
	rp.ID = 1
	if s.redPackets == nil {
		s.redPackets = map[int64]*RedPacketRecord{}
	}
	cloned := *rp
	s.redPackets[rp.ID] = &cloned
	return nil
}

func (s *redPacketRepoStub) GetByCode(context.Context, string) (*RedPacketRecord, error) {
	return nil, ErrRedPacketNotFound
}

func (s *redPacketRepoStub) GetByID(_ context.Context, id int64) (*RedPacketRecord, error) {
	rp, ok := s.redPackets[id]
	if !ok {
		return nil, ErrRedPacketNotFound
	}
	cloned := *rp
	return &cloned, nil
}

func (s *redPacketRepoStub) DecrementClaim(context.Context, int64, float64) error { return nil }

func (s *redPacketRepoStub) MarkExhausted(context.Context, int64) error { return nil }

func (s *redPacketRepoStub) MarkExpired(context.Context, int64) error { return nil }

func (s *redPacketRepoStub) CreateClaim(context.Context, *RedPacketClaimRecord) error { return nil }

func (s *redPacketRepoStub) HasClaimed(context.Context, int64, int64) (bool, error) {
	return false, nil
}

func (s *redPacketRepoStub) GetClaims(_ context.Context, redpacketID int64) ([]*RedPacketClaimRecord, error) {
	claims := s.claims[redpacketID]
	out := make([]*RedPacketClaimRecord, 0, len(claims))
	for _, claim := range claims {
		cloned := *claim
		out = append(out, &cloned)
	}
	return out, nil
}

func (s *redPacketRepoStub) ListBySender(context.Context, int64, int, int) ([]*RedPacketRecord, int, error) {
	return nil, 0, nil
}

func (s *redPacketRepoStub) ListActiveExpired(context.Context) ([]*RedPacketRecord, error) {
	out := make([]*RedPacketRecord, 0, len(s.activeExpired))
	for _, rp := range s.activeExpired {
		cloned := *rp
		out = append(out, &cloned)
	}
	return out, nil
}

func (s *redPacketRepoStub) ListAll(context.Context, int, int) ([]*RedPacketRecord, int, error) {
	return nil, 0, nil
}

func (s *redPacketRepoStub) ReturnRemaining(_ context.Context, id int64, _ int64) (float64, error) {
	if s.returnErr != nil {
		return 0, s.returnErr
	}
	return s.returnRemaining[id], nil
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
	records := make([]*BalanceTransferRecord, 0, len(s.listAllRecords))
	for _, record := range s.listAllRecords {
		cloned := *record
		records = append(records, &cloned)
	}
	return records, s.listAllTotal, nil
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

func newRedPacketTestService(
	t *testing.T,
	repo *transferRepoStub,
	redPacketRepo *redPacketRepoStub,
	userRepo *transferUserRepoStub,
	overrides map[string]string,
) *BalanceTransferService {
	t.Helper()

	settings := map[string]string{
		SettingKeyTransferEnabled:         "true",
		SettingKeyTransferFeeRate:         "0",
		SettingKeyTransferMinAmount:       "1",
		SettingKeyTransferMaxAmount:       "1000",
		SettingKeyRedPacketEnabled:        "true",
		SettingKeyRedPacketMaxCount:       "10",
		SettingKeyRedPacketExpireHours:    "24",
		SettingKeyTransferDailyLimit:      "0",
		SettingKeyTransferVIPFeeExempt:    "false",
		SettingKeyTransferDailyCountLimit: "0",
	}
	for key, value := range overrides {
		settings[key] = value
	}

	settingSvc := NewSettingService(&transferSettingRepoStub{settings: settings}, &config.Config{})
	return NewBalanceTransferService(repo, redPacketRepo, userRepo, settingSvc)
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

func TestCreateRedPacket_RechecksSenderBalanceInsideTx(t *testing.T) {
	repo := &transferRepoStub{
		lockedBalances: map[int64]float64{1: 20},
	}
	redPacketRepo := &redPacketRepoStub{}
	userRepo := &transferUserRepoStub{
		users: map[int64]*User{
			1: {ID: 1, Balance: 500},
		},
	}
	svc := newRedPacketTestService(t, repo, redPacketRepo, userRepo, nil)

	record, err := svc.CreateRedPacket(context.Background(), 1, 30, 2, "equal", nil)
	require.Nil(t, record)
	require.ErrorIs(t, err, ErrTransferInsufficient)
	require.Equal(t, 1, repo.runInTxCalls)
	require.False(t, redPacketRepo.created)
	require.Empty(t, userRepo.deductCalls)
}

func TestGetRedPacketDetail_OnlySenderCanViewSensitiveDetail(t *testing.T) {
	repo := &transferRepoStub{}
	redPacketRepo := &redPacketRepoStub{
		redPackets: map[int64]*RedPacketRecord{
			7: {
				ID:       7,
				SenderID: 1,
				Code:     "secret-code",
				Status:   "active",
			},
		},
		claims: map[int64][]*RedPacketClaimRecord{
			7: {{ID: 11, RedPacketID: 7, UserID: 2, Amount: 3}},
		},
	}
	userRepo := &transferUserRepoStub{
		users: map[int64]*User{
			1: {ID: 1, Email: "sender@example.com"},
			2: {ID: 2, Email: "claimant@example.com"},
		},
	}
	svc := newRedPacketTestService(t, repo, redPacketRepo, userRepo, nil)

	rp, claims, err := svc.GetRedPacketDetail(context.Background(), 1, 7)
	require.NoError(t, err)
	require.Equal(t, "secret-code", rp.Code)
	require.Len(t, claims, 1)
	require.Equal(t, "claimant@example.com", claims[0].UserEmail)

	rp, claims, err = svc.GetRedPacketDetail(context.Background(), 2, 7)
	require.Nil(t, rp)
	require.Nil(t, claims)
	require.ErrorIs(t, err, ErrRedPacketNotFound)
}

func TestExpireRedPackets_ReturnsRemainingOnceAndPropagatesErrors(t *testing.T) {
	repo := &transferRepoStub{}
	redPacketRepo := &redPacketRepoStub{
		activeExpired: []*RedPacketRecord{
			{ID: 7, SenderID: 1, RemainingAmount: 5, Status: "active"},
			{ID: 8, SenderID: 1, RemainingAmount: 0, Status: "active"},
		},
		returnRemaining: map[int64]float64{7: 5, 8: 0},
	}
	userRepo := &transferUserRepoStub{
		users: map[int64]*User{
			1: {ID: 1, Balance: 10},
		},
	}
	svc := newRedPacketTestService(t, repo, redPacketRepo, userRepo, nil)

	err := svc.ExpireRedPackets(context.Background())
	require.NoError(t, err)
	require.Equal(t, 2, repo.runInTxCalls)
	require.Equal(t, []transferBalanceMutation{{userID: 1, amount: 5}}, userRepo.updateCalls)

	repo = &transferRepoStub{}
	sentinelErr := errors.New("return failed")
	redPacketRepo = &redPacketRepoStub{
		activeExpired: []*RedPacketRecord{
			{ID: 9, SenderID: 1, RemainingAmount: 5, Status: "active"},
		},
		returnErr: sentinelErr,
	}
	svc = newRedPacketTestService(t, repo, redPacketRepo, userRepo, nil)

	err = svc.ExpireRedPackets(context.Background())
	require.ErrorIs(t, err, sentinelErr)
	require.Contains(t, err.Error(), "expire red packet 9")
}

func TestGetAllTransfers_BatchesSenderAndReceiverEmails(t *testing.T) {
	repo := &transferRepoStub{
		listAllTotal: 2,
		listAllRecords: []*BalanceTransferRecord{
			{ID: 1, SenderID: 1, ReceiverID: 2, Amount: 10},
			{ID: 2, SenderID: 1, ReceiverID: 3, Amount: 20},
		},
	}
	userRepo := &transferUserRepoStub{
		users: map[int64]*User{
			1: {ID: 1, Email: "sender@example.com"},
			2: {ID: 2, Email: "first@example.com"},
			3: {ID: 3, Email: "second@example.com"},
		},
	}
	svc := newTransferTestService(t, repo, userRepo, nil)

	records, total, err := svc.GetAllTransfers(context.Background(), nil, 1, 10)
	require.NoError(t, err)

	require.Equal(t, 2, total)
	require.Equal(t, []int64{1, 2, 3}, userRepo.batchEmailIDs)
	require.Zero(t, userRepo.getByIDCallCount)
	require.Equal(t, "sender@example.com", records[0].SenderEmail)
	require.Equal(t, "first@example.com", records[0].ReceiverEmail)
	require.Equal(t, "sender@example.com", records[1].SenderEmail)
	require.Equal(t, "second@example.com", records[1].ReceiverEmail)
}

func TestSearchUsers_RequiresMinimumLengthAndMasksResults(t *testing.T) {
	repo := &transferRepoStub{}
	userRepo := &transferUserRepoStub{
		listUsers: []User{
			{ID: 1, Email: "alice@example.com", Username: "alice"},
			{ID: 2, Email: "hidden@example.com", Username: "bob"},
		},
	}
	svc := newTransferTestService(t, repo, userRepo, nil)

	results, err := svc.SearchUsers(context.Background(), "a")
	require.NoError(t, err)
	require.Empty(t, results)
	require.Empty(t, userRepo.listFilters)

	results, err = svc.SearchUsers(context.Background(), "ali")
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, int64(1), results[0].ID)
	require.Equal(t, "a***e@e*****e.com", results[0].Email)
	require.Equal(t, "a***e", results[0].Username)
	require.Equal(t, "ali", userRepo.listFilters[0].Search)
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
