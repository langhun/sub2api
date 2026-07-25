// Package custom contains the application-specific Overlay extension points.
package custom

import (
	"database/sql"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	activitycheckin "github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/checkin"
	activityleaderboard "github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/leaderboard"
	activityredpacket "github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/redpacket"
	activityrewards "github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/rewards"
	gamehall "github.com/Wei-Shaw/sub2api/internal/custom/modules/game-hall"
	walletextension "github.com/Wei-Shaw/sub2api/internal/custom/modules/wallet-extension"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/wire"
)

// ProviderSet constructs the Overlay runtime at the composition root.
var ProviderSet = wire.NewSet(ProvideRuntime)

// Runtime owns dependencies shared by custom modules as they are introduced.
type Runtime struct {
	ActivityCheckin     *activitycheckin.Module
	ActivityLeaderboard *activityleaderboard.Module
	ActivityRedPacket   *activityredpacket.Module
	ActivityRewards     *activityrewards.Runtime
	ActivityRewardsHTTP *activityrewards.Module
	GameHall            *gamehall.Module
	WalletExtension     *walletextension.Module
}

// NewRuntime constructs the enabled custom modules at the composition root.
func NewRuntime(
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
	gameHallService := gamehall.NewGameHallService(gamehall.NewGameHallRepository(client, db), settingService, billingCache)
	activityRedPacketService := activityredpacket.NewService(activityredpacket.Dependencies{
		Repository:   activityredpacket.NewRepository(client),
		Transactions: activityredpacket.NewTransactionRunner(client),
		Balance:      activityredpacket.NewBalanceWriter(client),
		Settings:     activityredpacket.NewSettingsAdapter(settingService),
		Code:         activityredpacket.NewSettingsCodeGenerator(settingService),
		Fees:         activityredpacket.NewFeeAdapter(settingService, subscriptionService),
		Ledger:       activityredpacket.NewClaimLedger(client),
	})
	activityRedPacket := activityredpacket.NewModule(
		activityRedPacketService,
		activityredpacket.NewExpiryWorker(activityredpacket.ExpiryWorkerDependencies{
			Expire: activityRedPacketService,
			Leases: activityredpacket.NewLeaseCoordinator(leaderLockCache, db),
		}),
	)
	activityRewardsHTTP := activityrewards.NewLegacyModule(blindboxService)
	walletService := walletextension.NewService(
		walletextension.NewDirectTransferRepository(client),
		walletextension.NewSettingsAdapter(settingService),
		userRepository,
		subscriptionService,
		billingCache,
	)
	return &Runtime{
		ActivityCheckin:     activitycheckin.NewLegacyModule(checkinService, blindboxService),
		ActivityLeaderboard: activityleaderboard.NewLegacyModule(settingService, leaderboardService),
		ActivityRedPacket:   activityRedPacket,
		ActivityRewards:     activityrewards.NewLegacyRuntime(rewardDeliveryStore, blindboxService, activityrewards.WorkerOptions{}),
		ActivityRewardsHTTP: activityRewardsHTTP,
		GameHall:            gamehall.NewModule(gameHallService),
		WalletExtension:     walletextension.NewModule(walletService, legacyTransferService),
	}
}
