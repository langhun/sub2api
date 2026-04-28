//go:build unit

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type mockBalanceTransferRepo struct {
	createFn                func(ctx context.Context, t *BalanceTransferRecord) error
	getByIDFn               func(ctx context.Context, id int64) (*BalanceTransferRecord, error)
	updateStatusFn          func(ctx context.Context, id int64, status string, frozenAt *time.Time, frozenBy *int64, revokeReason *string) error
	listByUserFn            func(ctx context.Context, userID int64, role string, page, pageSize int) ([]*BalanceTransferRecord, int, error)
	listByUserExcludeTypeFn func(ctx context.Context, userID int64, role, excludeType string, page, pageSize int) ([]*BalanceTransferRecord, int, error)
	listAllFn               func(ctx context.Context, filter *TransferFilter, page, pageSize int) ([]*BalanceTransferRecord, int, error)
	getDailyTransferTotalFn func(ctx context.Context, userID int64) (float64, int, error)
	getFeeStatsFn           func(ctx context.Context, startTime, endTime time.Time) ([]*DailyFeeStat, error)
	getLeaderboardFn        func(ctx context.Context, startTime, endTime time.Time, limit int, orderBy string) ([]*TransferRankEntry, error)
	runInTxFn               func(ctx context.Context, fn func(ctx context.Context) error) error
	getUserTransferStatsFn  func(ctx context.Context, userID int64) (sent, received, feePaid float64, err error)
	createCalls             int
	runInTxCalls            int
}

func (m *mockBalanceTransferRepo) Create(ctx context.Context, t *BalanceTransferRecord) error {
	m.createCalls++
	if m.createFn != nil {
		return m.createFn(ctx, t)
	}
	return nil
}

func (m *mockBalanceTransferRepo) GetByID(ctx context.Context, id int64) (*BalanceTransferRecord, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	panic("unexpected GetByID call")
}

func (m *mockBalanceTransferRepo) UpdateStatus(ctx context.Context, id int64, status string, frozenAt *time.Time, frozenBy *int64, revokeReason *string) error {
	if m.updateStatusFn != nil {
		return m.updateStatusFn(ctx, id, status, frozenAt, frozenBy, revokeReason)
	}
	panic("unexpected UpdateStatus call")
}

func (m *mockBalanceTransferRepo) ListByUser(ctx context.Context, userID int64, role string, page, pageSize int) ([]*BalanceTransferRecord, int, error) {
	if m.listByUserFn != nil {
		return m.listByUserFn(ctx, userID, role, page, pageSize)
	}
	panic("unexpected ListByUser call")
}

func (m *mockBalanceTransferRepo) ListByUserExcludeType(ctx context.Context, userID int64, role, excludeType string, page, pageSize int) ([]*BalanceTransferRecord, int, error) {
	if m.listByUserExcludeTypeFn != nil {
		return m.listByUserExcludeTypeFn(ctx, userID, role, excludeType, page, pageSize)
	}
	panic("unexpected ListByUserExcludeType call")
}

func (m *mockBalanceTransferRepo) ListAll(ctx context.Context, filter *TransferFilter, page, pageSize int) ([]*BalanceTransferRecord, int, error) {
	if m.listAllFn != nil {
		return m.listAllFn(ctx, filter, page, pageSize)
	}
	panic("unexpected ListAll call")
}

func (m *mockBalanceTransferRepo) GetDailyTransferTotal(ctx context.Context, userID int64) (float64, int, error) {
	if m.getDailyTransferTotalFn != nil {
		return m.getDailyTransferTotalFn(ctx, userID)
	}
	panic("unexpected GetDailyTransferTotal call")
}

func (m *mockBalanceTransferRepo) GetFeeStats(ctx context.Context, startTime, endTime time.Time) ([]*DailyFeeStat, error) {
	if m.getFeeStatsFn != nil {
		return m.getFeeStatsFn(ctx, startTime, endTime)
	}
	panic("unexpected GetFeeStats call")
}

func (m *mockBalanceTransferRepo) GetLeaderboard(ctx context.Context, startTime, endTime time.Time, limit int, orderBy string) ([]*TransferRankEntry, error) {
	if m.getLeaderboardFn != nil {
		return m.getLeaderboardFn(ctx, startTime, endTime, limit, orderBy)
	}
	panic("unexpected GetLeaderboard call")
}

func (m *mockBalanceTransferRepo) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	m.runInTxCalls++
	if m.runInTxFn != nil {
		return m.runInTxFn(ctx, fn)
	}
	return fn(ctx)
}

func (m *mockBalanceTransferRepo) GetUserTransferStats(ctx context.Context, userID int64) (sent, received, feePaid float64, err error) {
	if m.getUserTransferStatsFn != nil {
		return m.getUserTransferStatsFn(ctx, userID)
	}
	panic("unexpected GetUserTransferStats call")
}

type mockBalanceRedPacketRepo struct {
	createFn            func(ctx context.Context, rp *RedPacketRecord) error
	getByCodeFn         func(ctx context.Context, code string) (*RedPacketRecord, error)
	getByIDFn           func(ctx context.Context, id int64) (*RedPacketRecord, error)
	decrementClaimFn    func(ctx context.Context, id int64, amount float64) error
	markExhaustedFn     func(ctx context.Context, id int64) error
	markExpiredFn       func(ctx context.Context, id int64) error
	createClaimFn       func(ctx context.Context, claim *RedPacketClaimRecord) error
	hasClaimedFn        func(ctx context.Context, redpacketID, userID int64) (bool, error)
	getClaimsFn         func(ctx context.Context, redpacketID int64) ([]*RedPacketClaimRecord, error)
	listBySenderFn      func(ctx context.Context, senderID int64, page, pageSize int) ([]*RedPacketRecord, int, error)
	listActiveExpiredFn func(ctx context.Context) ([]*RedPacketRecord, error)
	listAllFn           func(ctx context.Context, page, pageSize int) ([]*RedPacketRecord, int, error)
	returnRemainingFn   func(ctx context.Context, id int64, senderID int64) (float64, error)
	hasClaimedCalls     int
	getByIDCalls        int
	createClaimCalls    int
}

func (m *mockBalanceRedPacketRepo) Create(ctx context.Context, rp *RedPacketRecord) error {
	if m.createFn != nil {
		return m.createFn(ctx, rp)
	}
	panic("unexpected Create call")
}

func (m *mockBalanceRedPacketRepo) GetByCode(ctx context.Context, code string) (*RedPacketRecord, error) {
	if m.getByCodeFn != nil {
		return m.getByCodeFn(ctx, code)
	}
	panic("unexpected GetByCode call")
}

func (m *mockBalanceRedPacketRepo) GetByID(ctx context.Context, id int64) (*RedPacketRecord, error) {
	m.getByIDCalls++
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	panic("unexpected GetByID call")
}

func (m *mockBalanceRedPacketRepo) DecrementClaim(ctx context.Context, id int64, amount float64) error {
	if m.decrementClaimFn != nil {
		return m.decrementClaimFn(ctx, id, amount)
	}
	panic("unexpected DecrementClaim call")
}

func (m *mockBalanceRedPacketRepo) MarkExhausted(ctx context.Context, id int64) error {
	if m.markExhaustedFn != nil {
		return m.markExhaustedFn(ctx, id)
	}
	panic("unexpected MarkExhausted call")
}

func (m *mockBalanceRedPacketRepo) MarkExpired(ctx context.Context, id int64) error {
	if m.markExpiredFn != nil {
		return m.markExpiredFn(ctx, id)
	}
	panic("unexpected MarkExpired call")
}

func (m *mockBalanceRedPacketRepo) CreateClaim(ctx context.Context, claim *RedPacketClaimRecord) error {
	m.createClaimCalls++
	if m.createClaimFn != nil {
		return m.createClaimFn(ctx, claim)
	}
	panic("unexpected CreateClaim call")
}

func (m *mockBalanceRedPacketRepo) HasClaimed(ctx context.Context, redpacketID, userID int64) (bool, error) {
	m.hasClaimedCalls++
	if m.hasClaimedFn != nil {
		return m.hasClaimedFn(ctx, redpacketID, userID)
	}
	panic("unexpected HasClaimed call")
}

func (m *mockBalanceRedPacketRepo) GetClaims(ctx context.Context, redpacketID int64) ([]*RedPacketClaimRecord, error) {
	if m.getClaimsFn != nil {
		return m.getClaimsFn(ctx, redpacketID)
	}
	panic("unexpected GetClaims call")
}

func (m *mockBalanceRedPacketRepo) ListBySender(ctx context.Context, senderID int64, page, pageSize int) ([]*RedPacketRecord, int, error) {
	if m.listBySenderFn != nil {
		return m.listBySenderFn(ctx, senderID, page, pageSize)
	}
	panic("unexpected ListBySender call")
}

func (m *mockBalanceRedPacketRepo) ListActiveExpired(ctx context.Context) ([]*RedPacketRecord, error) {
	if m.listActiveExpiredFn != nil {
		return m.listActiveExpiredFn(ctx)
	}
	panic("unexpected ListActiveExpired call")
}

func (m *mockBalanceRedPacketRepo) ListAll(ctx context.Context, page, pageSize int) ([]*RedPacketRecord, int, error) {
	if m.listAllFn != nil {
		return m.listAllFn(ctx, page, pageSize)
	}
	panic("unexpected ListAll call")
}

func (m *mockBalanceRedPacketRepo) ReturnRemaining(ctx context.Context, id int64, senderID int64) (float64, error) {
	if m.returnRemainingFn != nil {
		return m.returnRemainingFn(ctx, id, senderID)
	}
	panic("unexpected ReturnRemaining call")
}

type mockTransferSettingRepo struct {
	values    map[string]string
	getAllErr error
}

func (m *mockTransferSettingRepo) Get(context.Context, string) (*Setting, error) {
	panic("unexpected Get call")
}

func (m *mockTransferSettingRepo) GetValue(context.Context, string) (string, error) {
	panic("unexpected GetValue call")
}

func (m *mockTransferSettingRepo) Set(context.Context, string, string) error {
	panic("unexpected Set call")
}

func (m *mockTransferSettingRepo) GetMultiple(context.Context, []string) (map[string]string, error) {
	panic("unexpected GetMultiple call")
}

func (m *mockTransferSettingRepo) SetMultiple(context.Context, map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (m *mockTransferSettingRepo) GetAll(context.Context) (map[string]string, error) {
	if m.getAllErr != nil {
		return nil, m.getAllErr
	}
	return m.values, nil
}

func (m *mockTransferSettingRepo) Delete(context.Context, string) error {
	panic("unexpected Delete call")
}

func newBalanceTransferTestService(transferRepo BalanceTransferRepository, redPacketRepo BalanceRedPacketRepository, userRepo UserRepository) *BalanceTransferService {
	settingRepo := &mockTransferSettingRepo{
		values: map[string]string{
			SettingKeyTransferEnabled:         "true",
			SettingKeyTransferFeeRate:         "0.01",
			SettingKeyTransferMinAmount:       "0.01",
			SettingKeyTransferMaxAmount:       "1000",
			SettingKeyTransferDailyLimit:      "1000",
			SettingKeyTransferDailyCountLimit: "50",
			SettingKeyRedPacketEnabled:        "true",
			SettingKeyRedPacketMaxCount:       "100",
			SettingKeyRedPacketExpireHours:    "24",
		},
	}
	cfg := &config.Config{
		Default: config.DefaultConfig{
			UserBalance:     0,
			UserConcurrency: 1,
		},
	}
	return NewBalanceTransferService(transferRepo, redPacketRepo, userRepo, NewSettingService(settingRepo, cfg))
}

func TestClaimRedPacketRecheckPreventsDuplicateWithinProcess(t *testing.T) {
	redPacketRepo := &mockBalanceRedPacketRepo{}
	transferRepo := &mockBalanceTransferRepo{}
	redPacketRepo.getByCodeFn = func(context.Context, string) (*RedPacketRecord, error) {
		return &RedPacketRecord{
			ID:              7,
			SenderID:        10,
			Status:          "active",
			ExpireAt:        time.Now().Add(time.Hour),
			RemainingAmount: 8,
			RemainingCount:  2,
			RedPacketType:   "equal",
		}, nil
	}
	redPacketRepo.hasClaimedFn = func(_ context.Context, _ int64, _ int64) (bool, error) {
		if redPacketRepo.hasClaimedCalls == 1 {
			return false, nil
		}
		return true, nil
	}
	svc := newBalanceTransferTestService(transferRepo, redPacketRepo, &mockUserRepo{})

	claim, err := svc.ClaimRedPacket(context.Background(), 99, "dup-check")
	require.Nil(t, claim)
	require.ErrorIs(t, err, ErrRedPacketAlreadyClaimed)
	require.Equal(t, 2, redPacketRepo.hasClaimedCalls)
	require.Zero(t, redPacketRepo.getByIDCalls)
	require.Zero(t, transferRepo.runInTxCalls)
	require.Zero(t, redPacketRepo.createClaimCalls)
}

func TestClaimRedPacketConstraintErrorMapsToAlreadyClaimed(t *testing.T) {
	transferRepo := &mockBalanceTransferRepo{
		createFn: func(_ context.Context, t *BalanceTransferRecord) error {
			t.ID = 88
			return nil
		},
	}
	redPacketRepo := &mockBalanceRedPacketRepo{
		getByCodeFn: func(context.Context, string) (*RedPacketRecord, error) {
			return &RedPacketRecord{
				ID:              9,
				SenderID:        10,
				Status:          "active",
				ExpireAt:        time.Now().Add(time.Hour),
				RemainingAmount: 10,
				RemainingCount:  2,
				RedPacketType:   "equal",
			}, nil
		},
		hasClaimedFn: func(context.Context, int64, int64) (bool, error) {
			return false, nil
		},
		getByIDFn: func(context.Context, int64) (*RedPacketRecord, error) {
			return &RedPacketRecord{
				ID:              9,
				SenderID:        10,
				Status:          "active",
				ExpireAt:        time.Now().Add(time.Hour),
				RemainingAmount: 10,
				RemainingCount:  2,
				RedPacketType:   "equal",
			}, nil
		},
		decrementClaimFn: func(context.Context, int64, float64) error {
			return nil
		},
		createClaimFn: func(context.Context, *RedPacketClaimRecord) error {
			return &dbent.ConstraintError{}
		},
	}
	userRepo := &mockUserRepo{
		updateBalanceFn: func(context.Context, int64, float64) error {
			return nil
		},
	}
	svc := newBalanceTransferTestService(transferRepo, redPacketRepo, userRepo)

	claim, err := svc.ClaimRedPacket(context.Background(), 99, "race")
	require.Nil(t, claim)
	require.ErrorIs(t, err, ErrRedPacketAlreadyClaimed)
	require.Equal(t, 2, redPacketRepo.hasClaimedCalls)
	require.Equal(t, 1, redPacketRepo.getByIDCalls)
	require.Equal(t, 1, transferRepo.runInTxCalls)
	require.Equal(t, 1, transferRepo.createCalls)
	require.Equal(t, 1, redPacketRepo.createClaimCalls)
}

func TestExpireRedPacketsReturnsJoinedError(t *testing.T) {
	redPacketRepo := &mockBalanceRedPacketRepo{
		listActiveExpiredFn: func(context.Context) ([]*RedPacketRecord, error) {
			return []*RedPacketRecord{
				{ID: 11, SenderID: 101},
				{ID: 22, SenderID: 202},
			}, nil
		},
		returnRemainingFn: func(_ context.Context, id int64, _ int64) (float64, error) {
			switch id {
			case 11:
				return 0, errors.New("db fail A")
			case 22:
				return 0, errors.New("db fail B")
			default:
				return 0, errors.New("db fail")
			}
		},
	}
	transferRepo := &mockBalanceTransferRepo{
		runInTxFn: func(ctx context.Context, fn func(ctx context.Context) error) error {
			return fn(ctx)
		},
	}
	svc := newBalanceTransferTestService(transferRepo, redPacketRepo, &mockUserRepo{})

	err := svc.ExpireRedPackets(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "expire red packet 11")
	require.Contains(t, err.Error(), "db fail A")
	require.Contains(t, err.Error(), "expire red packet 22")
	require.Contains(t, err.Error(), "db fail B")
	require.Equal(t, 2, transferRepo.runInTxCalls)
}
