package custom

import (
	"context"
	"database/sql"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// ProvideRuntime starts module-owned background workers after composition.
func ProvideRuntime(
	client *dbent.Client,
	db *sql.DB,
	settingService *service.SettingService,
	billingCache *service.BillingCacheService,
	userRepository service.UserRepository,
	subscriptionService *service.SubscriptionService,
	legacyTransferService *service.BalanceTransferService,
	checkinService *service.CheckinService,
	blindboxService *service.BlindBoxService,
	leaderboardService *service.LeaderboardService,
	rewardDeliveryStore service.RewardDeliveryStore,
	leaderLockCache service.LeaderLockCache,
) *Runtime {
	runtime := NewRuntime(
		client,
		db,
		settingService,
		billingCache,
		userRepository,
		subscriptionService,
		legacyTransferService,
		checkinService,
		blindboxService,
		leaderboardService,
		rewardDeliveryStore,
		leaderLockCache,
	)
	runtime.Start()
	return runtime
}

// Start launches the module-owned background workers once dependencies exist.
func (r *Runtime) Start() {
	if r == nil {
		return
	}
	if r.ActivityRedPacket != nil {
		r.ActivityRedPacket.Start()
	}
	if r.ActivityRewards != nil {
		r.ActivityRewards.Start(context.Background())
	}
}

// Stop waits for every module-owned background worker to finish.
func (r *Runtime) Stop() {
	if r == nil {
		return
	}
	if r.ActivityRewards != nil {
		r.ActivityRewards.Stop()
	}
	if r.ActivityRedPacket != nil {
		r.ActivityRedPacket.Stop()
	}
}
