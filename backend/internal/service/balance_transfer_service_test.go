//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type transferSafetyRepo struct {
	BalanceTransferRepository
	record              *BalanceTransferRecord
	dailyTotal          float64
	dailyCount          int
	deductOK            bool
	lockCalls           int
	dailyCalls          int
	deductCalls         int
	createCalls         int
	updateStatusCalls   int
	deductAfterLock     bool
	dailyAfterLock      bool
	lastCreditedUserID  int64
	lastCreditedBalance float64
}

func (r *transferSafetyRepo) RunInTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func (r *transferSafetyRepo) LockUser(context.Context, int64) error {
	r.lockCalls++
	return nil
}

func (r *transferSafetyRepo) GetDailyTransferTotal(context.Context, int64) (float64, int, error) {
	r.dailyCalls++
	r.dailyAfterLock = r.lockCalls > 0
	return r.dailyTotal, r.dailyCount, nil
}

func (r *transferSafetyRepo) DeductBalanceIfSufficient(context.Context, int64, float64) (bool, error) {
	r.deductCalls++
	r.deductAfterLock = r.lockCalls > 0
	return r.deductOK, nil
}

func (r *transferSafetyRepo) Create(_ context.Context, record *BalanceTransferRecord) error {
	r.createCalls++
	record.ID = 91
	return nil
}

func (r *transferSafetyRepo) GetByIDForUpdate(context.Context, int64) (*BalanceTransferRecord, error) {
	copy := *r.record
	return &copy, nil
}

func (r *transferSafetyRepo) UpdateStatus(context.Context, int64, string, *time.Time, *int64, *string) error {
	r.updateStatusCalls++
	return nil
}

type redPacketSafetyRepo struct {
	BalanceRedPacketRepository
	locked          *RedPacketRecord
	updated         *RedPacketRecord
	decrementAmount float64
	markCalls       int
	claimCalls      int
}

func (r *redPacketSafetyRepo) GetByCode(context.Context, string) (*RedPacketRecord, error) {
	copy := *r.locked
	copy.RemainingAmount = 99 // Deliberately stale pre-transaction snapshot.
	copy.RemainingCount = 99
	return &copy, nil
}

func (r *redPacketSafetyRepo) GetByCodeForUpdate(context.Context, string) (*RedPacketRecord, error) {
	copy := *r.locked
	return &copy, nil
}

func (r *redPacketSafetyRepo) DecrementClaim(_ context.Context, _ int64, amount float64) (*RedPacketRecord, error) {
	r.decrementAmount = amount
	copy := *r.updated
	return &copy, nil
}

func (r *redPacketSafetyRepo) CreateClaim(_ context.Context, claim *RedPacketClaimRecord) error {
	r.claimCalls++
	claim.ID = 72
	return nil
}

func (r *redPacketSafetyRepo) MarkExhausted(context.Context, int64) error {
	r.markCalls++
	return nil
}

type transferSafetyUserRepo struct {
	UserRepository
	creditCalls int
}

type transferSafetySettingRepo struct {
	values map[string]string
}

func (r *transferSafetySettingRepo) Get(context.Context, string) (*Setting, error) {
	return nil, ErrSettingNotFound
}
func (r *transferSafetySettingRepo) GetValue(context.Context, string) (string, error) {
	return "", ErrSettingNotFound
}
func (r *transferSafetySettingRepo) Set(context.Context, string, string) error { return nil }
func (r *transferSafetySettingRepo) GetMultiple(context.Context, []string) (map[string]string, error) {
	return r.values, nil
}
func (r *transferSafetySettingRepo) SetMultiple(context.Context, map[string]string) error { return nil }
func (r *transferSafetySettingRepo) GetAll(context.Context) (map[string]string, error) {
	return r.values, nil
}
func (r *transferSafetySettingRepo) Delete(context.Context, string) error { return nil }

func (r *transferSafetyUserRepo) GetByID(_ context.Context, id int64) (*User, error) {
	return &User{ID: id, Balance: 100}, nil
}

func (r *transferSafetyUserRepo) UpdateBalance(context.Context, int64, float64) error {
	r.creditCalls++
	return nil
}

func newTransferSafetySettings(values map[string]string) *SettingService {
	return NewSettingService(&transferSafetySettingRepo{values: values}, &config.Config{})
}

func TestBalanceTransferLocksBeforeLimitsAndRejectsInsufficientBalance(t *testing.T) {
	repo := &transferSafetyRepo{deductOK: false}
	users := &transferSafetyUserRepo{}
	svc := NewBalanceTransferService(repo, &redPacketSafetyRepo{}, users, newTransferSafetySettings(map[string]string{
		SettingKeyTransferEnabled:   "true",
		SettingKeyTransferMinAmount: "0.01",
	}), nil)

	_, err := svc.Transfer(context.Background(), 1, 2, 10, nil)

	require.ErrorIs(t, err, ErrTransferInsufficient)
	require.Equal(t, 1, repo.lockCalls)
	require.Equal(t, 1, repo.dailyCalls)
	require.True(t, repo.dailyAfterLock)
	require.True(t, repo.deductAfterLock)
	require.Zero(t, repo.createCalls)
	require.Zero(t, users.creditCalls)
}

func TestBalanceTransferEnforcesDailyCountInsideLockedTransaction(t *testing.T) {
	repo := &transferSafetyRepo{dailyCount: 1, deductOK: true}
	users := &transferSafetyUserRepo{}
	svc := NewBalanceTransferService(repo, &redPacketSafetyRepo{}, users, newTransferSafetySettings(map[string]string{
		SettingKeyTransferEnabled:         "true",
		SettingKeyTransferMinAmount:       "0.01",
		SettingKeyTransferDailyCountLimit: "1",
	}), nil)

	_, err := svc.Transfer(context.Background(), 1, 2, 10, nil)

	require.ErrorIs(t, err, ErrTransferDailyCount)
	require.True(t, repo.dailyAfterLock)
	require.Zero(t, repo.deductCalls)
	require.Zero(t, users.creditCalls)
}

func TestBalanceTransferRevokeIsIdempotent(t *testing.T) {
	repo := &transferSafetyRepo{record: &BalanceTransferRecord{ID: 8, Status: "revoked", SenderID: 1, ReceiverID: 2, Amount: 3, GrossAmount: 4}}
	users := &transferSafetyUserRepo{}
	svc := NewBalanceTransferService(repo, &redPacketSafetyRepo{}, users, newTransferSafetySettings(nil), nil)

	require.NoError(t, svc.RevokeTransfer(context.Background(), 10, 8, "retry"))
	require.Zero(t, repo.deductCalls)
	require.Zero(t, users.creditCalls)
	require.Zero(t, repo.updateStatusCalls)
}

func TestBalanceTransferClaimUsesLockedStateAndUpdatedExhaustion(t *testing.T) {
	packet := &RedPacketRecord{
		ID: 5, SenderID: 1, RemainingAmount: 2, RemainingCount: 2,
		RedPacketType: "equal", Status: "active", ExpireAt: time.Now().Add(time.Hour),
	}
	redRepo := &redPacketSafetyRepo{
		locked:  packet,
		updated: &RedPacketRecord{ID: 5, RemainingAmount: 1, RemainingCount: 0},
	}
	transferRepo := &transferSafetyRepo{deductOK: true}
	users := &transferSafetyUserRepo{}
	svc := NewBalanceTransferService(transferRepo, redRepo, users, newTransferSafetySettings(map[string]string{
		SettingKeyTransferEnabled:  "true",
		SettingKeyRedPacketEnabled: "true",
	}), nil)

	claim, err := svc.ClaimRedPacket(context.Background(), 2, "code")

	require.NoError(t, err)
	require.NotNil(t, claim)
	require.Equal(t, 1.0, redRepo.decrementAmount)
	require.Equal(t, 1, redRepo.claimCalls)
	require.Equal(t, 1, redRepo.markCalls)
	require.Equal(t, 1, users.creditCalls)
}
