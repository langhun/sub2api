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
	bepusdt "github.com/Wei-Shaw/sub2api/internal/custom/modules/bepusdt"
	codeformat "github.com/Wei-Shaw/sub2api/internal/custom/modules/code-format"
	docsmodule "github.com/Wei-Shaw/sub2api/internal/custom/modules/docs"
	gamehall "github.com/Wei-Shaw/sub2api/internal/custom/modules/game-hall"
	openairt "github.com/Wei-Shaw/sub2api/internal/custom/modules/openai-rt"
	rechargepromotions "github.com/Wei-Shaw/sub2api/internal/custom/modules/recharge-promotions"
	registrationprotection "github.com/Wei-Shaw/sub2api/internal/custom/modules/registration-protection"
	supportcontacts "github.com/Wei-Shaw/sub2api/internal/custom/modules/support-contacts"
	supporttickets "github.com/Wei-Shaw/sub2api/internal/custom/modules/support-tickets"
	"github.com/Wei-Shaw/sub2api/internal/custom/platform"
	customsettings "github.com/Wei-Shaw/sub2api/internal/custom/settings"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
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
	SupportTickets      *supporttickets.Module
	Docs                *docsmodule.Module
	OpenAIRT            *openairt.Module
	RegistrationRisk    *registrationprotection.Module
	RechargePromotions  *rechargepromotions.Module
	SupportContacts     *supportcontacts.Module
	UsageQuerySettings  UsageQuerySettings
}

// NewRuntime constructs the enabled custom modules at the composition root.
func NewRuntime(
	client *dbent.Client,
	db *sql.DB,
	activityWalletCapabilities *platform.ActivityWalletCapabilities,
	redeemService *service.RedeemService,
	adminService service.AdminService,
	openaiOAuthService *service.OpenAIOAuthService,
	tokenCacheInvalidator service.TokenCacheInvalidator,
	customSettingsRegistry *customsettings.Registry,
	codeGenerator *codeformat.Generator,
	paymentService *service.PaymentService,
	authService *service.AuthService,
	apiKeyService *service.APIKeyService,
	redisClient *redis.Client,
) (*Runtime, error) {
	bepusdt.Register()
	if err := custommigrations.Apply(context.Background(), db); err != nil {
		return nil, err
	}
	service.ConfigureCodeGenerator(redeemService, adminService, codeGenerator)
	registrationRiskStore := registrationprotection.NewStore(db)
	registrationPolicy := registrationprotection.NewPolicy(customSettingsRegistry, registrationRiskStore)
	registrationPolicy.SetRedis(redisClient)
	registrationRiskStoreForAdmin := registrationprotection.NewRiskStore(db, customSettingsRegistry)
	registrationRiskStoreForAdmin.SetRedis(redisClient)
	registrationRiskModule := registrationprotection.NewModule(db, registrationRiskStoreForAdmin)
	signupBalanceLedger := registrationprotection.NewSignupBalanceLedger(db)
	if authService != nil {
		authService.SetRegistrationEmailPolicy(registrationPolicy)
		authService.SetRegistrationRiskPolicy(registrationPolicy)
		authService.SetSignupBalanceLedger(signupBalanceLedger)
	}
	if historyRegistrar, ok := adminService.(interface {
		SetSignupBalanceHistoryProvider(service.SignupBalanceHistoryProvider)
	}); ok {
		historyRegistrar.SetSignupBalanceHistoryProvider(signupBalanceLedger)
	}
	if apiKeyService != nil {
		apiKeyService.SetCreationPolicy(registrationPolicy)
		apiKeyService.SetRequestPolicy(registrationPolicy)
	}
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
		Fees:         activityredpacket.NewZeroFeeAdapter(),
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
	checkinProtector, err := newCheckinProtector(authService, redisClient)
	if err != nil {
		return nil, err
	}
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
		RewardEligibility: registrationPolicy,
	}, activitycheckin.NewEntBlindboxRecordsReader(client), checkinProtector)
	if err != nil {
		return nil, err
	}
	rechargePromotions := rechargepromotions.NewModule(customSettingsRegistry, db)
	if paymentService != nil {
		paymentService.SetBalanceOrderExtension(rechargePromotions)
	}
	return &Runtime{
		ActivityCheckin:     activityCheckin,
		ActivityLeaderboard: activityleaderboard.NewDatabaseModule(client, db),
		ActivityRedPacket:   activityRedPacket,
		ActivityRewards:     activityRewards,
		ActivityRewardsHTTP: activityRewardsHTTP,
		GameHall:            gamehall.NewModuleWithIdempotency(gameHallService, platform.DefaultIdempotencyCoordinator()),
		SupportTickets:      supporttickets.NewModule(supporttickets.NewService(supporttickets.NewRepository(db), customSettingsRegistry)),
		Docs:                docsmodule.NewModule(docsmodule.NewService(docsmodule.NewRepository(db))),
		OpenAIRT:            openairt.NewModule(adminService, openaiOAuthService, tokenCacheInvalidator),
		RegistrationRisk:    registrationRiskModule,
		RechargePromotions:  rechargePromotions,
		SupportContacts:     supportcontacts.NewModule(db),
		UsageQuerySettings:  customSettingsRegistry,
	}, nil
}
