package walletextension

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/custom/modules/wallet-extension/contract"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestServiceTransferCommitsDirectPlanAndInvalidatesBothBalances(t *testing.T) {
	store := &directTransferStoreStub{record: DirectTransferRecord{ID: 12, TransferType: DirectTransferType, Status: "completed"}}
	cache := &balanceCacheStub{}
	svc := NewService(store, directTransferSettingsStub{settings: enabledDirectTransferSettings()}, directTransferAccountStub{accounts: map[int64]contract.Account{
		2: {ID: 2, Status: accountStatusActive, Username: "receiver", Email: "receiver@example.com"},
	}}, nil, nil, nil, cache)

	record, err := svc.Transfer(context.Background(), 1, DirectTransferRequest{ReceiverID: 2, Amount: 10})
	require.NoError(t, err)
	require.Equal(t, int64(12), record.ID)
	require.Equal(t, 10.0, store.plan.Amount)
	require.Equal(t, 0.1, store.plan.Fee)
	require.Equal(t, 10.1, store.plan.GrossAmount)
	require.Equal(t, []int64{1, 2}, cache.invalidated)
}

func TestServicePreviewRejectsDailyLimitBeforeCommit(t *testing.T) {
	store := &directTransferStoreStub{dailyAmount: 95}
	svc := NewService(store, directTransferSettingsStub{settings: enabledDirectTransferSettings()}, directTransferAccountStub{accounts: map[int64]contract.Account{
		2: {ID: 2, Status: accountStatusActive, Username: "receiver", Email: "receiver@example.com"},
	}}, nil, nil, nil, nil)

	_, err := svc.Preview(context.Background(), 1, 2, 10)
	require.ErrorIs(t, err, ErrTransferDailyLimit)
	require.Equal(t, 0, store.commitCalls)
}

func TestServiceResolveRecipientMasksIdentity(t *testing.T) {
	resolved := contract.Recipient{Account: contract.Account{ID: 2, Status: accountStatusActive, Username: "alice", Email: "alice@example.com"}, DisplayName: "alice"}
	svc := NewService(&directTransferStoreStub{}, directTransferSettingsStub{settings: enabledDirectTransferSettings()}, nil, directTransferRecipientStub{resolved: resolved}, nil, nil, nil)

	recipient, err := svc.ResolveRecipient(context.Background(), 1, "alice@example.com")
	require.NoError(t, err)
	require.Equal(t, int64(2), recipient.Account.ID)
	require.Equal(t, "alice", recipient.DisplayName)
}

func TestServiceRevokeBatchDoesNotCreditAdministrator(t *testing.T) {
	store := &directTransferStoreStub{
		adminRecord: TransferRecord{ID: 8, Status: "completed", TransferType: batchTransferType, SenderID: 10, ReceiverID: 2, Amount: 3, GrossAmount: 3},
		debitOK:     true,
	}
	balances := &directTransferBalanceStub{}
	svc := NewService(store, nil, nil, nil, nil, balances, nil)

	require.NoError(t, svc.RevokeTransfer(context.Background(), 10, 8, "rollback grant"))
	require.Equal(t, 1, store.debitCalls)
	require.Equal(t, 1, store.updateStatusCalls)
	require.Empty(t, balances.adjustments)
}

func TestServiceRevokeIsIdempotent(t *testing.T) {
	store := &directTransferStoreStub{adminRecord: TransferRecord{ID: 8, Status: "revoked", SenderID: 1, ReceiverID: 2, Amount: 3, GrossAmount: 4}}
	balances := &directTransferBalanceStub{}
	svc := NewService(store, nil, nil, nil, nil, balances, nil)

	require.NoError(t, svc.RevokeTransfer(context.Background(), 10, 8, "retry"))
	require.Zero(t, store.debitCalls)
	require.Zero(t, store.updateStatusCalls)
	require.Empty(t, balances.adjustments)
}

func TestServiceBatchDistributeRecordsModuleOwnedLedgerEntries(t *testing.T) {
	store := &directTransferStoreStub{nextTransferID: 91}
	accounts := directTransferAccountStub{accounts: map[int64]contract.Account{2: {ID: 2, Status: accountStatusActive}}}
	balances := &directTransferBalanceStub{}
	svc := NewService(store, nil, accounts, nil, nil, balances, nil)

	items, err := svc.BatchDistribute(context.Background(), 10, []BatchDistributeTarget{{UserID: 2, Amount: 6}, {UserID: 0, Amount: 9}}, nil)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, int64(91), items[0].ID)
	require.Equal(t, batchTransferType, items[0].TransferType)
	require.Equal(t, []balanceAdjustment{{UserID: 2, Amount: 6}}, balances.adjustments)
}

func TestServiceLeaderboardUsesModuleRepository(t *testing.T) {
	store := &directTransferStoreStub{leaderboard: []TransferRankEntry{{Rank: 1, UserID: 7, TotalAmount: 12}}}
	settings := directTransferSettingsStub{
		settings:    enabledDirectTransferSettings(),
		leaderboard: TransferLeaderboardSettings{Enabled: true},
	}
	svc := NewService(store, settings, nil, nil, nil, nil, nil)

	entries, err := svc.GetLeaderboard(context.Background(), "week", 20)
	require.NoError(t, err)
	require.Equal(t, store.leaderboard, entries)
}

func TestModuleRegistersOnlyDirectTransferRoutes(t *testing.T) {
	module := NewModule(nil)
	router := newWalletExtensionRouter(module)
	routes := make(map[string]struct{})
	for _, route := range router.Routes() {
		routes[route.Method+" "+route.Path] = struct{}{}
	}
	for _, route := range []string{
		"POST /api/v1/transfer",
		"POST /api/v1/transfer/validate",
		"GET /api/v1/transfer/receiver",
		"GET /api/v1/transfer/receivers",
		"GET /api/v1/transfer/history",
		"GET /api/v1/transfer/stats",
		"GET /api/v1/transfer/leaderboard",
		"GET /api/v1/admin/transfers",
		"GET /api/v1/admin/transfers/stats",
		"PUT /api/v1/admin/transfers/:id/freeze",
		"PUT /api/v1/admin/transfers/:id/revoke",
		"POST /api/v1/admin/transfers/batch",
	} {
		_, exists := routes[route]
		require.Truef(t, exists, "missing route %s", route)
	}
	_, exists := routes["GET /api/v1/admin/redpackets"]
	require.False(t, exists, "red packets remain activity-owned")
}

func enabledDirectTransferSettings() contract.Settings {
	return contract.Settings{DirectTransfer: contract.DirectTransferSettings{
		Enabled: true, FeeRate: 0.01, MinimumAmount: 1, MaximumAmount: 100, DailyLimit: 100, DailyCountLimit: 5,
	}}
}

type directTransferStoreStub struct {
	plan              DirectTransferCommitPlan
	record            DirectTransferRecord
	dailyAmount       float64
	dailyCount        int
	commitCalls       int
	adminRecord       TransferRecord
	adminList         []TransferRecord
	debitOK           bool
	debitCalls        int
	updateStatusCalls int
	nextTransferID    int64
	createdTransfers  []TransferRecord
	leaderboard       []TransferRankEntry
}

func (s *directTransferStoreStub) CommitDirectTransfer(_ context.Context, plan DirectTransferCommitPlan) (DirectTransferRecord, error) {
	s.plan = plan
	s.commitCalls++
	return s.record, nil
}
func (s *directTransferStoreStub) CreateDirectTransfer(context.Context, *DirectTransferRecord) error {
	return nil
}
func (s *directTransferStoreStub) GetDirectTransfer(context.Context, int64) (DirectTransferRecord, error) {
	return DirectTransferRecord{}, nil
}
func (s *directTransferStoreStub) ListDirectTransferHistory(context.Context, DirectTransferHistoryQuery) ([]DirectTransferRecord, int, error) {
	return nil, 0, nil
}
func (s *directTransferStoreStub) GetDirectTransferDailyUsage(context.Context, int64, time.Time, time.Time) (float64, int, error) {
	return s.dailyAmount, s.dailyCount, nil
}
func (s *directTransferStoreStub) GetDirectTransferStats(context.Context, int64) (DirectTransferStats, error) {
	return DirectTransferStats{}, nil
}
func (s *directTransferStoreStub) ListAllTransfers(context.Context, TransferFilter, int, int) ([]TransferRecord, int, error) {
	return s.adminList, len(s.adminList), nil
}
func (s *directTransferStoreStub) GetTransferForUpdate(context.Context, int64) (TransferRecord, error) {
	return s.adminRecord, nil
}
func (s *directTransferStoreStub) UpdateTransferStatus(_ context.Context, _ int64, status string, _ *time.Time, _ *int64, _ *string) error {
	s.updateStatusCalls++
	s.adminRecord.Status = status
	return nil
}
func (s *directTransferStoreStub) DebitBalanceIfSufficient(context.Context, int64, float64) (bool, error) {
	s.debitCalls++
	return s.debitOK, nil
}
func (s *directTransferStoreStub) CreateTransfer(_ context.Context, record *TransferRecord) error {
	if s.nextTransferID == 0 {
		s.nextTransferID = 1
	}
	record.ID = s.nextTransferID
	s.nextTransferID++
	s.createdTransfers = append(s.createdTransfers, *record)
	return nil
}
func (s *directTransferStoreStub) GetTransferFeeStats(context.Context, time.Time, time.Time) ([]DailyFeeStat, error) {
	return nil, nil
}
func (s *directTransferStoreStub) GetTransferLeaderboard(context.Context, time.Time, time.Time, int) ([]TransferRankEntry, error) {
	return s.leaderboard, nil
}
func (s *directTransferStoreStub) RunInTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type directTransferSettingsStub struct {
	settings    contract.Settings
	leaderboard TransferLeaderboardSettings
}

func (s directTransferSettingsStub) GetWalletExtensionSettings(context.Context) (contract.Settings, error) {
	return s.settings, nil
}
func (s directTransferSettingsStub) GetWalletTransferLeaderboardSettings(context.Context) (TransferLeaderboardSettings, error) {
	return s.leaderboard, nil
}

type directTransferAccountStub struct{ accounts map[int64]contract.Account }

func (s directTransferAccountStub) GetAccount(_ context.Context, id int64) (contract.Account, error) {
	return s.accounts[id], nil
}

type directTransferRecipientStub struct{ resolved contract.Recipient }

func (s directTransferRecipientStub) ResolveDirectTransferRecipient(context.Context, int64, string) (contract.Recipient, error) {
	return s.resolved, nil
}

func (directTransferRecipientStub) SearchDirectTransferRecipients(context.Context, int64, string, int) ([]contract.RecipientCandidate, error) {
	return nil, nil
}

type directTransferBalanceStub struct{ adjustments []balanceAdjustment }

func (s *directTransferBalanceStub) Credit(_ context.Context, operation contract.BalanceOperation) error {
	s.adjustments = append(s.adjustments, balanceAdjustment{UserID: operation.AccountID, Amount: operation.Amount})
	return nil
}

func (*directTransferBalanceStub) DebitIfSufficient(context.Context, contract.BalanceOperation) (bool, error) {
	return false, nil
}

type balanceAdjustment struct {
	UserID int64
	Amount float64
}

type balanceCacheStub struct{ invalidated []int64 }

func (s *balanceCacheStub) InvalidateBalance(_ context.Context, userID int64) error {
	s.invalidated = append(s.invalidated, userID)
	return nil
}

func newWalletExtensionRouter(module *Module) *gin.Engine {
	router := gin.New()
	passthrough := func(c *gin.Context) { c.Next() }
	module.RegisterRoutes(
		router,
		middleware.JWTAuthMiddleware(passthrough),
		middleware.AdminAuthMiddleware(passthrough),
		middleware.AuditLogMiddleware(passthrough),
		nil,
	)
	return router
}
