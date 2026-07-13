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
	claims          []*RedPacketClaimRecord
	getClaimsErr    error
	lookupCodes     []string
}

func (r *redPacketSafetyRepo) GetByCode(_ context.Context, code string) (*RedPacketRecord, error) {
	r.lookupCodes = append(r.lookupCodes, code)
	copy := *r.locked
	copy.RemainingAmount = 99 // Deliberately stale pre-transaction snapshot.
	copy.RemainingCount = 99
	return &copy, nil
}

func (r *redPacketSafetyRepo) GetByCodeForUpdate(_ context.Context, code string) (*RedPacketRecord, error) {
	r.lookupCodes = append(r.lookupCodes, code)
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

func (r *redPacketSafetyRepo) GetByID(context.Context, int64) (*RedPacketRecord, error) {
	copy := *r.locked
	return &copy, nil
}

func (r *redPacketSafetyRepo) GetClaims(context.Context, int64) ([]*RedPacketClaimRecord, error) {
	return r.claims, r.getClaimsErr
}

func TestClaimRedPacketNormalizesCodeBeforeBothLookups(t *testing.T) {
	now := time.Now()
	redPacketRepo := &redPacketSafetyRepo{
		locked: &RedPacketRecord{
			ID: 2, SenderID: 10, RemainingAmount: 1, RemainingCount: 1,
			RedPacketType: "equal", Status: "active", ExpireAt: now.Add(time.Hour),
		},
		updated: &RedPacketRecord{ID: 2, RemainingAmount: 0, RemainingCount: 0},
	}
	transferRepo := &transferSafetyRepo{deductOK: true}
	userRepo := &transferSafetyUserRepo{}
	settingService := newTransferSafetySettings(map[string]string{
		SettingKeyRedPacketEnabled: "true",
	})
	service := NewBalanceTransferService(transferRepo, redPacketRepo, userRepo, settingService, nil)

	_, err := service.ClaimRedPacket(context.Background(), 20, "  rp-Lower-ABC  ")

	require.NoError(t, err)
	require.Equal(t, []string{"rp-Lower-ABC", "rp-Lower-ABC"}, redPacketRepo.lookupCodes)
}

type transferSafetyUserRepo struct {
	UserRepository
	creditCalls             int
	nonRechargeCreditCalls  int
	userByID                map[int64]*User
	conditionalBalanceOK    bool
	conditionalBalanceCalls int
	resolvedReceiver        *User
	resolveErr              error
	resolveQuery            string
	resolveNumericID        *int64
	searchedReceivers       []*User
	searchErr               error
	searchQuery             string
	searchRequesterID       int64
	searchLimit             int
}

func (r *transferSafetyUserRepo) ResolveActiveTransferReceiver(_ context.Context, query string, numericID *int64) (*User, error) {
	r.resolveQuery = query
	if numericID != nil {
		id := *numericID
		r.resolveNumericID = &id
	}
	return r.resolvedReceiver, r.resolveErr
}

func (r *transferSafetyUserRepo) SearchActiveTransferReceivers(_ context.Context, query string, requesterID int64, limit int) ([]*User, error) {
	r.searchQuery = query
	r.searchRequesterID = requesterID
	r.searchLimit = limit
	return r.searchedReceivers, r.searchErr
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
	if user := r.userByID[id]; user != nil {
		return user, nil
	}
	return &User{ID: id, Balance: 100}, nil
}

func (r *transferSafetyUserRepo) UpdateBalance(context.Context, int64, float64) error {
	r.creditCalls++
	return nil
}

func (r *transferSafetyUserRepo) UpdateBalanceWithoutRecharge(context.Context, int64, float64) error {
	r.nonRechargeCreditCalls++
	return nil
}

func (r *transferSafetyUserRepo) UpdateBalanceWithoutRechargeIfNonnegative(context.Context, int64, float64) (bool, error) {
	r.conditionalBalanceCalls++
	if r.conditionalBalanceOK {
		r.nonRechargeCreditCalls++
	}
	return r.conditionalBalanceOK, nil
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

func TestBalanceTransferResolveReceiverSupportsNumericIDAndExactIdentity(t *testing.T) {
	settings := newTransferSafetySettings(map[string]string{SettingKeyTransferEnabled: "true"})

	t.Run("numeric id bypasses minimum text length", func(t *testing.T) {
		users := &transferSafetyUserRepo{resolvedReceiver: &User{ID: 7, Status: StatusActive, Username: "alice"}}
		svc := NewBalanceTransferService(&transferSafetyRepo{}, &redPacketSafetyRepo{}, users, settings, nil)
		result, err := svc.ResolveReceiver(context.Background(), 1, "7")
		require.NoError(t, err)
		require.Equal(t, int64(7), result.ReceiverID)
		require.Equal(t, "alice", result.ReceiverDisplay)
		require.NotNil(t, users.resolveNumericID)
		require.Equal(t, int64(7), *users.resolveNumericID)
	})

	t.Run("exact email is passed without fuzzy expansion", func(t *testing.T) {
		users := &transferSafetyUserRepo{resolvedReceiver: &User{ID: 8, Status: StatusActive, Email: "alice@example.com"}}
		svc := NewBalanceTransferService(&transferSafetyRepo{}, &redPacketSafetyRepo{}, users, settings, nil)
		result, err := svc.ResolveReceiver(context.Background(), 1, " Alice@Example.com ")
		require.NoError(t, err)
		require.Equal(t, "Alice@Example.com", users.resolveQuery)
		require.Nil(t, users.resolveNumericID)
		require.Equal(t, "alice@example.com", result.ReceiverDisplay)
	})
}

func TestBalanceTransferResolveReceiverRejectsUnsafeQueriesAndExcludedUsers(t *testing.T) {
	settings := newTransferSafetySettings(map[string]string{SettingKeyTransferEnabled: "true"})
	tests := map[string]struct {
		query       string
		requesterID int64
		resolved    *User
		wantErr     error
	}{
		"empty":            {query: "", requesterID: 1, wantErr: ErrTransferReceiverQueryInvalid},
		"one character":    {query: "a", requesterID: 1, wantErr: ErrTransferReceiverQueryInvalid},
		"numeric zero":     {query: "0", requesterID: 1, wantErr: ErrTransferReceiverQueryInvalid},
		"numeric overflow": {query: "999999999999999999999999", requesterID: 1, wantErr: ErrTransferReceiverQueryInvalid},
		"self":             {query: "self", requesterID: 4, resolved: &User{ID: 4, Status: StatusActive}, wantErr: ErrTransferReceiverNotFound},
		"disabled":         {query: "disabled", requesterID: 1, resolved: &User{ID: 5, Status: StatusDisabled}, wantErr: ErrTransferReceiverNotFound},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			users := &transferSafetyUserRepo{resolvedReceiver: tc.resolved}
			svc := NewBalanceTransferService(&transferSafetyRepo{}, &redPacketSafetyRepo{}, users, settings, nil)
			_, err := svc.ResolveReceiver(context.Background(), tc.requesterID, tc.query)
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestBalanceTransferSearchReceiversReturnsMaskedCandidates(t *testing.T) {
	settings := newTransferSafetySettings(map[string]string{SettingKeyTransferEnabled: "true"})
	users := &transferSafetyUserRepo{searchedReceivers: []*User{
		{ID: 2, Status: StatusActive, Username: "openGate", Email: "identity@domain.icu"},
		{ID: 1, Status: StatusActive, Username: "requester", Email: "self@example.com"},
		{ID: 3, Status: StatusDisabled, Username: "disabled", Email: "disabled@example.com"},
		{ID: 4, Status: StatusActive, Email: "fallback@example.com"},
	}}
	svc := NewBalanceTransferService(&transferSafetyRepo{}, &redPacketSafetyRepo{}, users, settings, nil)

	result, err := svc.SearchReceivers(context.Background(), 1, "  gate  ")

	require.NoError(t, err)
	require.Equal(t, "gate", users.searchQuery)
	require.Equal(t, int64(1), users.searchRequesterID)
	require.Equal(t, 8, users.searchLimit)
	require.Len(t, result, 2)
	require.Equal(t, &TransferReceiverCandidate{
		ReceiverID:       2,
		ReceiverDisplay:  "o******e",
		ReceiverUsername: "o******e",
		ReceiverEmail:    "i******y@d****n.icu",
	}, result[0])
	require.Equal(t, "f******k@e*****e.com", result[1].ReceiverDisplay)
	require.Empty(t, result[1].ReceiverUsername)
	require.Equal(t, "f******k@e*****e.com", result[1].ReceiverEmail)
}

func TestBalanceTransferSearchReceiversValidatesQueryAndCapsResults(t *testing.T) {
	settings := newTransferSafetySettings(map[string]string{SettingKeyTransferEnabled: "true"})
	users := &transferSafetyUserRepo{}
	for id := int64(2); id <= 11; id++ {
		users.searchedReceivers = append(users.searchedReceivers, &User{
			ID: id, Status: StatusActive, Username: "candidate", Email: "candidate@example.com",
		})
	}
	svc := NewBalanceTransferService(&transferSafetyRepo{}, &redPacketSafetyRepo{}, users, settings, nil)

	_, err := svc.SearchReceivers(context.Background(), 1, "a")
	require.ErrorIs(t, err, ErrTransferReceiverQueryInvalid)

	result, err := svc.SearchReceivers(context.Background(), 1, "7")
	require.NoError(t, err)
	require.Len(t, result, 8)

	result, err = svc.SearchReceivers(context.Background(), 1, "候选")
	require.NoError(t, err)
	require.Len(t, result, 8)
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

func TestBalanceTransferDailyLimitUsesGrossAmount(t *testing.T) {
	settings := newTransferSafetySettings(map[string]string{
		SettingKeyTransferEnabled:    "true",
		SettingKeyTransferMinAmount:  "0.01",
		SettingKeyTransferFeeRate:    "0.1",
		SettingKeyTransferDailyLimit: "100",
	})

	t.Run("fee pushes gross amount over the limit", func(t *testing.T) {
		repo := &transferSafetyRepo{dailyTotal: 90, deductOK: true}
		users := &transferSafetyUserRepo{}
		svc := NewBalanceTransferService(repo, &redPacketSafetyRepo{}, users, settings, nil)

		_, err := svc.Transfer(context.Background(), 1, 2, 10, nil)

		require.ErrorIs(t, err, ErrTransferDailyLimit)
		require.True(t, repo.dailyAfterLock)
		require.Zero(t, repo.deductCalls)
		require.Zero(t, repo.createCalls)
		require.Zero(t, users.nonRechargeCreditCalls)
	})

	t.Run("gross amount exactly at the limit succeeds", func(t *testing.T) {
		repo := &transferSafetyRepo{dailyTotal: 89, deductOK: true}
		users := &transferSafetyUserRepo{}
		svc := NewBalanceTransferService(repo, &redPacketSafetyRepo{}, users, settings, nil)

		record, err := svc.Transfer(context.Background(), 1, 2, 10, nil)

		require.NoError(t, err)
		require.InDelta(t, 1, record.Fee, 1e-8)
		require.InDelta(t, 11, record.GrossAmount, 1e-8)
		require.Equal(t, 1, repo.deductCalls)
		require.Equal(t, 1, repo.createCalls)
		require.Equal(t, 1, users.nonRechargeCreditCalls)
	})
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

func TestBalanceTransferRevokeBatchDoesNotMintAdminBalance(t *testing.T) {
	repo := &transferSafetyRepo{record: &BalanceTransferRecord{ID: 8, Status: "completed", TransferType: "batch", SenderID: 10, ReceiverID: 2, Amount: 3, GrossAmount: 3}, deductOK: true}
	users := &transferSafetyUserRepo{conditionalBalanceOK: true}
	svc := NewBalanceTransferService(repo, &redPacketSafetyRepo{}, users, newTransferSafetySettings(nil), nil)

	require.NoError(t, svc.RevokeTransfer(context.Background(), 10, 8, "rollback grant"))
	require.Equal(t, 1, repo.deductCalls)
	require.Zero(t, users.nonRechargeCreditCalls)
	require.Equal(t, 1, repo.updateStatusCalls)
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
	require.Equal(t, 1, users.nonRechargeCreditCalls)
	require.Zero(t, users.creditCalls)
}

func TestGetRedPacketDetailForUserRequiresParticipation(t *testing.T) {
	packet := &RedPacketRecord{ID: 5, SenderID: 1}
	redRepo := &redPacketSafetyRepo{
		locked: packet,
		claims: []*RedPacketClaimRecord{
			{RedPacketID: 5, UserID: 2, Amount: 1},
			{RedPacketID: 5, UserID: 4, Amount: 2},
		},
	}
	svc := NewBalanceTransferService(&transferSafetyRepo{}, redRepo, &transferSafetyUserRepo{}, newTransferSafetySettings(map[string]string{
		SettingKeyRedPacketEnabled: "true",
	}), nil)

	_, _, err := svc.GetRedPacketDetailForUser(context.Background(), 3, 5)
	require.ErrorIs(t, err, ErrRedPacketDetailForbidden)
	_, claims, err := svc.GetRedPacketDetailForUser(context.Background(), 2, 5)
	require.NoError(t, err)
	require.Len(t, claims, 1)
	require.Equal(t, int64(2), claims[0].UserID)
	_, claims, err = svc.GetRedPacketDetailForUser(context.Background(), 1, 5)
	require.NoError(t, err)
	require.Len(t, claims, 2)
}

func TestGenerateRedPacketCodeUsesConfiguredDefaultShape(t *testing.T) {
	code, err := generateRedPacketCode()
	require.NoError(t, err)
	settings := DefaultCodeFormatSettings().RedPacket
	require.True(t, len(code) >= len(settings.Prefix))
	require.Equal(t, settings.Prefix, code[:len(settings.Prefix)])
}

func TestValidateTransferReturnsReceiverAndDailyPreview(t *testing.T) {
	repo := &transferSafetyRepo{dailyTotal: 25, dailyCount: 2}
	users := &transferSafetyUserRepo{userByID: map[int64]*User{
		2: {ID: 2, Username: "alice", Email: "alice@example.com"},
	}}
	svc := NewBalanceTransferService(repo, &redPacketSafetyRepo{}, users, newTransferSafetySettings(map[string]string{
		SettingKeyTransferEnabled:         "true",
		SettingKeyTransferMinAmount:       "0.01",
		SettingKeyTransferFeeRate:         "0.1",
		SettingKeyTransferDailyLimit:      "100",
		SettingKeyTransferDailyCountLimit: "5",
	}), nil)

	preview, err := svc.ValidateTransfer(context.Background(), 1, 2, 10)
	require.NoError(t, err)
	require.InDelta(t, 1, preview.Fee, 1e-8)
	require.InDelta(t, 11, preview.GrossAmount, 1e-8)
	require.Equal(t, "alice", preview.ReceiverDisplay)
	require.InDelta(t, 75, preview.DailyRemainingAmount, 1e-8)
	require.Equal(t, 3, preview.DailyRemainingCount)
}

func TestValidateTransferDailyLimitUsesGrossAmount(t *testing.T) {
	users := &transferSafetyUserRepo{userByID: map[int64]*User{
		2: {ID: 2, Username: "alice"},
	}}
	settings := newTransferSafetySettings(map[string]string{
		SettingKeyTransferEnabled:    "true",
		SettingKeyTransferMinAmount:  "0.01",
		SettingKeyTransferFeeRate:    "0.1",
		SettingKeyTransferDailyLimit: "100",
	})

	t.Run("fee pushes gross amount over the limit", func(t *testing.T) {
		repo := &transferSafetyRepo{dailyTotal: 90}
		svc := NewBalanceTransferService(repo, &redPacketSafetyRepo{}, users, settings, nil)

		_, err := svc.ValidateTransfer(context.Background(), 1, 2, 10)

		require.ErrorIs(t, err, ErrTransferDailyLimit)
	})

	t.Run("gross amount exactly at the limit succeeds", func(t *testing.T) {
		repo := &transferSafetyRepo{dailyTotal: 89, dailyCount: 2}
		svc := NewBalanceTransferService(repo, &redPacketSafetyRepo{}, users, settings, nil)

		preview, err := svc.ValidateTransfer(context.Background(), 1, 2, 10)

		require.NoError(t, err)
		require.InDelta(t, 11, preview.GrossAmount, 1e-8)
		require.InDelta(t, 11, preview.DailyRemainingAmount, 1e-8)
	})
}

func TestTransferLeaderboardRequiresLeaderboardSwitches(t *testing.T) {
	tests := []struct {
		name     string
		settings map[string]string
	}{
		{
			name: "leaderboard master disabled",
			settings: map[string]string{
				SettingKeyTransferEnabled:            "true",
				SettingKeyLeaderboardEnabled:         "false",
				SettingKeyLeaderboardTransferEnabled: "true",
			},
		},
		{
			name: "transfer board disabled",
			settings: map[string]string{
				SettingKeyTransferEnabled:            "true",
				SettingKeyLeaderboardEnabled:         "true",
				SettingKeyLeaderboardTransferEnabled: "false",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewBalanceTransferService(
				&transferSafetyRepo{},
				&redPacketSafetyRepo{},
				&transferSafetyUserRepo{},
				newTransferSafetySettings(tt.settings),
				nil,
			)

			_, err := svc.GetLeaderboard(context.Background(), "day", 20)

			require.ErrorIs(t, err, ErrTransferLeaderboardDisabled)
		})
	}
}

func TestConditionalInternalDebitRejectsNegativeResult(t *testing.T) {
	users := &transferSafetyUserRepo{conditionalBalanceOK: false}
	updated, err := updateBalanceWithoutRechargeIfNonnegative(context.Background(), users, 1, -10)
	require.NoError(t, err)
	require.False(t, updated)
	require.Equal(t, 1, users.conditionalBalanceCalls)
	require.Zero(t, users.nonRechargeCreditCalls)
	require.Zero(t, users.creditCalls)
}

func TestLuckCheckinSettingIsIndependentFromNormalCheckin(t *testing.T) {
	settings := newTransferSafetySettings(map[string]string{
		SettingKeyCheckinEnabled:     "false",
		SettingKeyCheckinLuckEnabled: "true",
	})
	require.False(t, settings.IsCheckinEnabled(context.Background()))
	require.True(t, settings.IsCheckinLuckEnabled(context.Background()))
}
