// Package custom contains the application-specific Overlay extension points.
package custom

import (
	"context"
	"database/sql"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	activitycheckin "github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/checkin"
	activityleaderboard "github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/leaderboard"
	activityredpacket "github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/redpacket"
	activityrewards "github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/rewards"
	gamehall "github.com/Wei-Shaw/sub2api/internal/custom/modules/game-hall"
	walletextension "github.com/Wei-Shaw/sub2api/internal/custom/modules/wallet-extension"
	"github.com/Wei-Shaw/sub2api/internal/custom/platform"
	customsettings "github.com/Wei-Shaw/sub2api/internal/custom/settings"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/wire"
)

// ProviderSet constructs the Overlay runtime at the composition root.
var ProviderSet = wire.NewSet(customsettings.ProviderSet, ProvideRuntime)

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
	redeemCodeRepository service.RedeemCodeRepository,
	customSettingsRegistry *customsettings.Registry,
	leaderLockCache service.LeaderLockCache,
) (*Runtime, error) {
	gameHallService := gamehall.NewGameHallService(
		gamehall.NewGameHallRepository(client, db),
		gamehall.NewRegistrySettingsAdapter(customSettingsRegistry),
		billingCache,
	)
	activityRedPacketService := activityredpacket.NewService(activityredpacket.Dependencies{
		Repository:   activityredpacket.NewRepository(client),
		Transactions: activityredpacket.NewTransactionRunner(client),
		Balance:      activityredpacket.NewBalanceWriter(client),
		Settings:     activityredpacket.NewRegistrySettingsAdapter(customSettingsRegistry),
		Code:         activityredpacket.NewSettingsCodeGenerator(activityRedPacketCodeGenerator{settings: settingService}),
		Fees:         activityredpacket.NewRegistryFeeAdapter(customSettingsRegistry, subscriptionService),
		Ledger:       activityredpacket.NewClaimLedger(client),
	})
	activityRedPacket := activityredpacket.NewModuleWithIdempotency(
		activityRedPacketService,
		activityredpacket.NewExpiryWorker(activityredpacket.ExpiryWorkerDependencies{
			Expire: activityRedPacketService,
			Leases: activityredpacket.NewLeaseCoordinator(leaderLockCache, db),
		}),
		platform.DefaultIdempotencyCoordinator(),
	)
	activityRewardsHTTP, activityRewards := activityrewards.NewProduction(activityrewards.ProductionDependencies{
		Client:        client,
		DB:            db,
		Settings:      customSettingsRegistry,
		CodeGenerator: settingService,
		Users:         userRepository,
		Subscriptions: subscriptionService,
		RedeemCodes:   redeemCodeRepository,
		BillingCache:  billingCache,
	}, activityrewards.WorkerOptions{})
	activityCheckin, err := activitycheckin.NewOperationalModule(activitycheckin.Dependencies{
		Repository:   activitycheckin.NewRepository(client),
		Transactions: activitycheckin.NewTransactionRunner(client),
		Accounts:     activitycheckin.NewEntAccountReader(client),
		Balance:      activitycheckin.NewEntBalanceWriter(client),
		Ledger:       activitycheckin.NewEntCheckinLedger(client, activitycheckin.NewCodeFormatGenerator(settingService)),
		Settings:     activitycheckin.NewRegistrySettingsAdapter(customSettingsRegistry),
		Cache:        activitycheckin.NewBalanceCacheInvalidator(billingCache),
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
		walletextension.NewBalanceCacheInvalidator(billingCache),
	)
	return &Runtime{
		ActivityCheckin:     activityCheckin,
		ActivityLeaderboard: activityleaderboard.NewDatabaseModule(client, db),
		ActivityRedPacket:   activityRedPacket,
		ActivityRewards:     activityRewards,
		ActivityRewardsHTTP: activityRewardsHTTP,
		GameHall:            gamehall.NewModuleWithIdempotency(gameHallService, platform.DefaultIdempotencyCoordinator()),
		WalletExtension:     walletextension.NewModule(walletService),
	}, nil
}

// activityRedPacketCodeGenerator keeps the legacy configurable code format at
// the composition root. Activity receives only the one code-generation port.
type activityRedPacketCodeGenerator struct {
	settings *service.SettingService
}

func (g activityRedPacketCodeGenerator) GenerateRedPacketCode(ctx context.Context) (string, error) {
	formats := service.DefaultCodeFormatSettings()
	if g.settings != nil {
		formats = g.settings.GetCodeFormatSettings(ctx)
	}
	return formats.RedPacket.Generate()
}
