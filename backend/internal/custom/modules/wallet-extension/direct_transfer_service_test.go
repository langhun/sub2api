package walletextension

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/custom/modules/wallet-extension/contract"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestServiceTransferCommitsDirectPlanAndInvalidatesBothBalances(t *testing.T) {
	store := &directTransferStoreStub{record: DirectTransferRecord{ID: 12, TransferType: DirectTransferType, Status: "completed"}}
	cache := &balanceCacheStub{}
	svc := NewService(store, directTransferSettingsStub{settings: enabledDirectTransferSettings()}, directTransferUserStub{users: map[int64]*service.User{
		2: {ID: 2, Status: service.StatusActive, Username: "receiver", Email: "receiver@example.com"},
	}}, nil, cache)

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
	svc := NewService(store, directTransferSettingsStub{settings: enabledDirectTransferSettings()}, directTransferUserStub{users: map[int64]*service.User{
		2: {ID: 2, Status: service.StatusActive, Username: "receiver", Email: "receiver@example.com"},
	}}, nil, nil)

	_, err := svc.Preview(context.Background(), 1, 2, 10)
	require.ErrorIs(t, err, ErrTransferDailyLimit)
	require.Equal(t, 0, store.commitCalls)
}

func TestServiceResolveRecipientMasksIdentity(t *testing.T) {
	user := &service.User{ID: 2, Status: service.StatusActive, Username: "alice", Email: "alice@example.com"}
	svc := NewService(&directTransferStoreStub{}, directTransferSettingsStub{settings: enabledDirectTransferSettings()}, directTransferUserStub{users: map[int64]*service.User{2: user}, resolved: user}, nil, nil)

	recipient, err := svc.ResolveRecipient(context.Background(), 1, "alice@example.com")
	require.NoError(t, err)
	require.Equal(t, int64(2), recipient.Account.ID)
	require.Equal(t, "alice", recipient.DisplayName)
}

func TestModuleRegistersOnlyDirectTransferRoutes(t *testing.T) {
	module := NewModule(nil, nil)
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
	plan        DirectTransferCommitPlan
	record      DirectTransferRecord
	dailyAmount float64
	dailyCount  int
	commitCalls int
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

type directTransferSettingsStub struct{ settings contract.Settings }

func (s directTransferSettingsStub) GetWalletExtensionSettings(context.Context) (contract.Settings, error) {
	return s.settings, nil
}

type directTransferUserStub struct {
	users    map[int64]*service.User
	resolved *service.User
}

func (s directTransferUserStub) GetByID(_ context.Context, id int64) (*service.User, error) {
	return s.users[id], nil
}
func (s directTransferUserStub) ResolveActiveTransferReceiver(context.Context, string, *int64) (*service.User, error) {
	return s.resolved, nil
}
func (s directTransferUserStub) SearchActiveTransferReceivers(context.Context, string, int64, int) ([]*service.User, error) {
	return nil, nil
}

type balanceCacheStub struct{ invalidated []int64 }

func (s *balanceCacheStub) InvalidateUserBalance(_ context.Context, userID int64) error {
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
