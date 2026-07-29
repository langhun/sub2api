// Package custom contains the application-specific Overlay extension points.
package custom

import (
	"context"
	"database/sql"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	custommigrations "github.com/Wei-Shaw/sub2api/internal/custom/migrations"
	activitycheckin "github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/checkin"
	activityleaderboard "github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/leaderboard"
	activityredpacket "github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/redpacket"
	activityrewards "github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/rewards"
	codeformat "github.com/Wei-Shaw/sub2api/internal/custom/modules/code-format"
	gamehall "github.com/Wei-Shaw/sub2api/internal/custom/modules/game-hall"
	walletextension "github.com/Wei-Shaw/sub2api/internal/custom/modules/wallet-extension"
	"github.com/Wei-Shaw/sub2api/internal/custom/platform"
	customsettings "github.com/Wei-Shaw/sub2api/internal/custom/settings"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/wire"
)

// ProviderSet constructs the Overlay runtime at the composition root.
var ProviderSet = wire.NewSet(
	customsettings.ProviderSet,
	platform.ProvideActivityWalletCapabilities,
	codeformat.NewGenerator,
	ProvideRuntime,
)

// Runtime owns dependencies shared by custom modules as they are introduced.
type Runtime struct {
	ActivityCheckin     *activitycheckin.Module
	ActivityLeaderboard *activityleaderboard.Module
	ActivityRedPacket   *activityredpacket.Module
	ActivityRewards     *activityrewards.Runtime
	ActivityRewardsHTTP *activityrewards.Module
	GameHall            *gamehall.Module
	UsageQuerySettings  UsageQuerySettings
	WalletExtension     *walletextension.Module
}

// NewRuntime constructs the enabled custom modules at the composition root.
func NewRuntime(
	client *dbent.Client,
	db *sql.DB,
	activityWalletCapabilities *platform.ActivityWalletCapabilities,
	redeemService *service.RedeemService,
	adminService service.AdminService,
	customSettingsRegistry *customsettings.Registry,
	codeGenerator *codeformat.Generator,
) (*Runtime, error) {
	if err := custommigrations.Apply(context.Background(), db); err != nil {
		return nil, err
	}
	service.ConfigureCodeGenerator(redeemService, adminService, codeGenerator)
	var (
		balanceCache  platform.BalanceCacheInvalidator
		concurrency   platform.UserConcurrencyWriter
		subscriptions platform.SubscriptionManager
		redeemRecords platform.RedeemRecordWriter
		leaderLocks   platform.LeaderLockCache
	)
	if activityWalletCapabilities != nil {
		balanceCache = activityWalletCapabilities.BalanceCache
		concurrency = activityWalletCapabilities.Concurrency
		subscriptions = activityWalletCapabilities.Subscriptions
		redeemRecords = activityWalletCapabilities.RedeemRecords
		leaderLocks = activityWalletCapabilities.LeaderLocks
	}

	gameHallService := gamehall.NewGameHallService(
		gamehall.NewGameHallRepository(client, db),
		gamehall.NewRegistrySettingsAdapter(customSettingsRegistry),
		balanceCache,
	)
	activityRedPacketService := activityredpacket.NewService(activityredpacket.Dependencies{
		Repository:   activityredpacket.NewRepository(client),
		Transactions: activityredpacket.NewTransactionRunner(client),
		Balance:      activityredpacket.NewBalanceWriter(client),
		Settings:     activityredpacket.NewRegistrySettingsAdapter(customSettingsRegistry),
		Code:         activityredpacket.NewSettingsCodeGenerator(codeGenerator),
		Fees:         activityredpacket.NewRegistryFeeAdapter(customSettingsRegistry, subscriptions),
		Ledger:       activityredpacket.NewClaimLedger(client),
	})
	activityRedPacket := activityredpacket.NewModuleWithIdempotency(
		activityRedPacketService,
		activityredpacket.NewExpiryWorker(activityredpacket.ExpiryWorkerDependencies{
			Expire: activityRedPacketService,
			Leases: activityredpacket.NewLeaseCoordinator(leaderLocks, db),
		}),
		platform.DefaultIdempotencyCoordinator(),
	)
	activityRewardsHTTP, activityRewards := activityrewards.NewProduction(activityrewards.ProductionDependencies{
		Client:        client,
		DB:            db,
		Settings:      customSettingsRegistry,
		CodeGenerator: codeGenerator,
		Concurrency:   concurrency,
		Subscriptions: subscriptions,
		RedeemRecords: redeemRecords,
		BalanceCache:  balanceCache,
	}, activityrewards.WorkerOptions{})
	activityCheckin, err := activitycheckin.NewOperationalModule(activitycheckin.Dependencies{
		Repository:   activitycheckin.NewRepository(client),
		Transactions: activitycheckin.NewTransactionRunner(client),
		Accounts:     activitycheckin.NewEntAccountReader(client),
		Balance:      activitycheckin.NewEntBalanceWriter(client),
		Ledger:       activitycheckin.NewEntCheckinLedger(client, activitycheckin.NewCodeFormatGenerator(codeGenerator)),
		Settings:     activitycheckin.NewRegistrySettingsAdapter(customSettingsRegistry),
		Cache:        activitycheckin.NewBalanceCacheInvalidator(balanceCache),
		Blindbox: activitycheckin.NewRewardsBlindboxDelivery(
			activityRewardsHTTP.Rewards,
			activityRewardsHTTP.Runner,
			activityRewardsHTTP.Outbox,
		),
	}, activitycheckin.NewEntBlindboxRecordsReader(client))
	if err != nil {
		return nil, err
	}
	walletService := walletextension.NewService(
		walletextension.NewDirectTransferRepository(client),
		walletextension.NewRegistrySettingsAdapter(customSettingsRegistry),
		walletextension.NewEntAccountReader(client),
		walletextension.NewEntRecipientResolver(client),
		walletextension.NewEntActiveSubscriptionReader(client),
		walletextension.NewEntBalanceWriter(client),
		walletextension.NewBalanceCacheInvalidator(balanceCache),
	)
	return &Runtime{
		ActivityCheckin:     activityCheckin,
		ActivityLeaderboard: activityleaderboard.NewDatabaseModule(client, db),
		ActivityRedPacket:   activityRedPacket,
		ActivityRewards:     activityRewards,
		ActivityRewardsHTTP: activityRewardsHTTP,
		GameHall:            gamehall.NewModuleWithIdempotency(gameHallService, platform.DefaultIdempotencyCoordinator()),
		UsageQuerySettings:  customSettingsRegistry,
		WalletExtension:     walletextension.NewModuleWithIdempotency(walletService, platform.DefaultIdempotencyCoordinator()),
	}, nil
}
