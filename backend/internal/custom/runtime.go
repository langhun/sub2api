// Package custom contains the application-specific Overlay extension points.
package custom

import (
	"database/sql"

	dbent "github.com/Wei-Shaw/sub2api/ent"
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
	codeformat.NewGenerator,
	wire.Bind(new(service.CodeGenerator), new(*codeformat.Generator)),
	ProvideRedeemService,
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
	WalletExtension     *walletextension.Module
}

// NewRuntime constructs the enabled custom modules at the composition root.
func NewRuntime(
	client *dbent.Client,
	db *sql.DB,
	billingCache *service.BillingCacheService,
	userRepository service.UserRepository,
	subscriptionService *service.SubscriptionService,
	redeemCodeRepository service.RedeemCodeRepository,
	customSettingsRegistry *customsettings.Registry,
	leaderLockCache service.LeaderLockCache,
	codeGenerator *codeformat.Generator,
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
		Code:         activityredpacket.NewSettingsCodeGenerator(codeGenerator),
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
		CodeGenerator: codeGenerator,
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
		Ledger:       activitycheckin.NewEntCheckinLedger(client, activitycheckin.NewCodeFormatGenerator(codeGenerator)),
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

// ProvideRedeemService makes the core redemption workflow depend only on the
// Overlay-provided code-generation port.
func ProvideRedeemService(
	redeemRepo service.RedeemCodeRepository,
	userRepo service.UserRepository,
	subscriptionService *service.SubscriptionService,
	cache service.RedeemCache,
	billingCache *service.BillingCacheService,
	client *dbent.Client,
	authCacheInvalidator service.APIKeyAuthCacheInvalidator,
	affiliateService *service.AffiliateService,
	codeGenerator service.CodeGenerator,
) *service.RedeemService {
	return service.NewRedeemService(
		redeemRepo, userRepo, subscriptionService, cache, billingCache, client,
		authCacheInvalidator, affiliateService, codeGenerator,
	)
}
