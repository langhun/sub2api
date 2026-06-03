//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

type balanceUserRepoStub struct {
	*userRepoStub
	updateErr error
	updated   []*User
}

func (s *balanceUserRepoStub) Update(ctx context.Context, user *User) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	if user == nil {
		return nil
	}
	clone := *user
	s.updated = append(s.updated, &clone)
	if s.userRepoStub != nil {
		s.userRepoStub.user = &clone
	}
	return nil
}

type balanceRedeemRepoStub struct {
	*redeemRepoStub
	created []*RedeemCode
}

func (s *balanceRedeemRepoStub) Create(ctx context.Context, code *RedeemCode) error {
	if code == nil {
		return nil
	}
	clone := *code
	s.created = append(s.created, &clone)
	return nil
}

type authCacheInvalidatorStub struct {
	userIDs  []int64
	groupIDs []int64
	keys     []string
}

func (s *authCacheInvalidatorStub) InvalidateAuthCacheByKey(ctx context.Context, key string) {
	s.keys = append(s.keys, key)
}

func (s *authCacheInvalidatorStub) InvalidateAuthCacheByUserID(ctx context.Context, userID int64) {
	s.userIDs = append(s.userIDs, userID)
}

func (s *authCacheInvalidatorStub) InvalidateAuthCacheByGroupID(ctx context.Context, groupID int64) {
	s.groupIDs = append(s.groupIDs, groupID)
}

type adminBalanceAdjusterStub struct {
	req    AdminBalanceAdjustmentRequest
	result *TransferFundsResult
	err    error
}

func (s *adminBalanceAdjusterStub) ApplyAdminBalanceAdjustment(ctx context.Context, req AdminBalanceAdjustmentRequest) (*TransferFundsResult, error) {
	s.req = req
	if s.err != nil {
		return nil, s.err
	}
	if s.result != nil {
		return s.result, nil
	}
	return &TransferFundsResult{
		TxID:    uuid.New(),
		UserID:  req.UserID,
		Amount:  req.Amount,
		Balance: req.Amount,
	}, nil
}

func TestAdminService_UpdateUserBalance_InvalidatesAuthCache(t *testing.T) {
	baseRepo := &userRepoStub{user: &User{ID: 7, Balance: 10}}
	repo := &balanceUserRepoStub{userRepoStub: baseRepo}
	redeemRepo := &balanceRedeemRepoStub{redeemRepoStub: &redeemRepoStub{}}
	invalidator := &authCacheInvalidatorStub{}
	bank := &adminBalanceAdjusterStub{result: &TransferFundsResult{
		TxID:    uuid.New(),
		UserID:  7,
		Amount:  decimal.NewFromInt(5),
		Balance: decimal.NewFromInt(15),
	}}
	svc := &adminServiceImpl{
		userRepo:             repo,
		redeemCodeRepo:       redeemRepo,
		authCacheInvalidator: invalidator,
		bankService:          bank,
	}

	user, err := svc.UpdateUserBalance(context.Background(), 7, 5, "add", "", "idem-add")
	require.NoError(t, err)
	require.Equal(t, 15.0, user.Balance)
	require.Equal(t, int64(7), bank.req.UserID)
	require.Equal(t, "add", bank.req.Operation)
	require.Equal(t, "idem-add", bank.req.IdempotencyKey)
	require.True(t, decimal.NewFromInt(5).Equal(bank.req.Amount))
	require.Equal(t, []int64{7}, invalidator.userIDs)
	require.Len(t, redeemRepo.created, 1)
	require.Empty(t, repo.updated)
}

func TestAdminService_UpdateUserBalance_NoChangeNoInvalidate(t *testing.T) {
	baseRepo := &userRepoStub{user: &User{ID: 7, Balance: 10}}
	repo := &balanceUserRepoStub{userRepoStub: baseRepo}
	redeemRepo := &balanceRedeemRepoStub{redeemRepoStub: &redeemRepoStub{}}
	invalidator := &authCacheInvalidatorStub{}
	bank := &adminBalanceAdjusterStub{result: &TransferFundsResult{
		UserID:  7,
		Amount:  decimal.Zero,
		Balance: decimal.NewFromInt(10),
	}}
	svc := &adminServiceImpl{
		userRepo:             repo,
		redeemCodeRepo:       redeemRepo,
		authCacheInvalidator: invalidator,
		bankService:          bank,
	}

	user, err := svc.UpdateUserBalance(context.Background(), 7, 10, "set", "", "idem-set")
	require.NoError(t, err)
	require.Equal(t, 10.0, user.Balance)
	require.Empty(t, invalidator.userIDs)
	require.Empty(t, redeemRepo.created)
	require.Empty(t, repo.updated)
}
