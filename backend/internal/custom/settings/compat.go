package settings

import (
	"context"
	"fmt"

	applethemesettings "github.com/Wei-Shaw/sub2api/internal/custom/modules/apple-theme/settings"
	codeformatsettings "github.com/Wei-Shaw/sub2api/internal/custom/modules/code-format/settings"
	rechargepromotionsettings "github.com/Wei-Shaw/sub2api/internal/custom/modules/recharge-promotions/settings"
)

// Compatibility preserves the established flat admin settings contract while
// keeping the actual configuration split by Overlay module in Snapshot.
type Compatibility struct {
	CodeFormatSettings                             codeformatsettings.Config                        `json:"code_format_settings"`
	DefaultHomepage                                string                                           `json:"default_homepage"`
	ThemeStyle                                     string                                           `json:"theme_style"`
	EnableAppleTheme                               bool                                             `json:"enable_apple_theme"`
	DefaultThemeMode                               string                                           `json:"default_theme_mode"`
	GameHallEnabled                                bool                                             `json:"game_hall_enabled"`
	RegistrationAliasProtectionEnabled             bool                                             `json:"registration_alias_protection_enabled"`
	RegistrationRewardFingerprintProtectionEnabled bool                                             `json:"registration_reward_fingerprint_protection_enabled"`
	RegistrationClusterProtectionEnabled           bool                                             `json:"registration_cluster_protection_enabled"`
	RegistrationClusterWindow6h                    int                                              `json:"registration_cluster_window_6h"`
	RegistrationClusterWindow24h                   int                                              `json:"registration_cluster_window_24h"`
	RegistrationClusterMediumThreshold             int                                              `json:"registration_cluster_medium_threshold"`
	RegistrationClusterHighThreshold               int                                              `json:"registration_cluster_high_threshold"`
	RegistrationAutoHighRiskLockEnabled            bool                                             `json:"registration_auto_high_risk_lock_enabled"`
	RegistrationTrialEnabled                       bool                                             `json:"registration_trial_enabled"`
	RegistrationTrialHours                         int                                              `json:"registration_trial_hours"`
	RegistrationTrialCostLimit                     float64                                          `json:"registration_trial_cost_limit"`
	RegistrationTrialRPMLimit                      int                                              `json:"registration_trial_rpm_limit"`
	RegistrationTrialConcurrency                   int                                              `json:"registration_trial_concurrency"`
	RegistrationPaymentUnlockEnabled               bool                                             `json:"registration_payment_unlock_enabled"`
	RegistrationPaymentUnlockAmount                float64                                          `json:"registration_payment_unlock_amount"`
	SupportTicketEnabled                           bool                                             `json:"support_ticket_enabled"`
	GameSlotsEnabled                               bool                                             `json:"game_slots_enabled"`
	GameSlotsMinBet                                float64                                          `json:"game_slots_min_bet"`
	GameSlotsMaxBet                                float64                                          `json:"game_slots_max_bet"`
	GameExchangeMinAmount                          float64                                          `json:"game_exchange_min_amount"`
	GameExchangeMaxAmount                          float64                                          `json:"game_exchange_max_amount"`
	GameExchangeDailyLimit                         float64                                          `json:"game_exchange_daily_limit"`
	GameExchangeAllowDGToBalance                   bool                                             `json:"game_exchange_allow_dg_to_balance"`
	CheckinEnabled                                 bool                                             `json:"checkin_enabled"`
	CheckinMinBalance                              float64                                          `json:"checkin_min_balance"`
	CheckinMaxBalance                              float64                                          `json:"checkin_max_balance"`
	CheckinLuckEnabled                             bool                                             `json:"checkin_luck_enabled"`
	CheckinLuckMinMultiplier                       float64                                          `json:"checkin_luck_min_multiplier"`
	CheckinLuckMaxMultiplier                       float64                                          `json:"checkin_luck_max_multiplier"`
	CheckinBlindboxEnabled                         bool                                             `json:"checkin_blindbox_enabled"`
	CheckinBlindboxTriggerType                     string                                           `json:"checkin_blindbox_trigger_type"`
	CheckinBlindboxInterval                        int                                              `json:"checkin_blindbox_interval"`
	RedPacketEnabled                               bool                                             `json:"redpacket_enabled"`
	RedPacketMaxCount                              int                                              `json:"redpacket_max_count"`
	RedPacketExpireHours                           int                                              `json:"redpacket_expire_hours"`
	UsageQueryEnabled                              bool                                             `json:"usage_query_enabled"`
	LeaderboardEnabled                             bool                                             `json:"leaderboard_enabled"`
	LeaderboardBalanceEnabled                      bool                                             `json:"leaderboard_balance_enabled"`
	LeaderboardConsumptionEnabled                  bool                                             `json:"leaderboard_consumption_enabled"`
	LeaderboardCheckinEnabled                      bool                                             `json:"leaderboard_checkin_enabled"`
	LeaderboardIncludeAdmin                        bool                                             `json:"leaderboard_include_admin"`
	RechargePromotionEnabled                       bool                                             `json:"recharge_promotion_enabled"`
	RechargePromotionMode                          string                                           `json:"recharge_promotion_mode"`
	RechargePromotionPercentageRate                float64                                          `json:"recharge_promotion_percentage_rate"`
	RechargePromotionMinimumAmount                 float64                                          `json:"recharge_promotion_minimum_amount"`
	RechargePromotionMaximumBonus                  float64                                          `json:"recharge_promotion_maximum_bonus"`
	RechargePromotionTierRules                     []rechargepromotionsettings.TierRule             `json:"recharge_promotion_tier_rules"`
	RechargePromotionRateTiers                     []rechargepromotionsettings.TieredPercentageRule `json:"recharge_promotion_tiered_percentage_rules"`
	RechargePromotionStartAt                       string                                           `json:"recharge_promotion_start_at"`
	RechargePromotionEndAt                         string                                           `json:"recharge_promotion_end_at"`
}

// PublicCompatibility is the intentionally smaller public projection. Values
// such as transfer limits and check-in amounts remain administrator-only.
type PublicCompatibility struct {
	DefaultHomepage                 string                                           `json:"default_homepage"`
	ThemeStyle                      string                                           `json:"theme_style"`
	EnableAppleTheme                bool                                             `json:"enable_apple_theme"`
	DefaultThemeMode                string                                           `json:"default_theme_mode"`
	GameHallEnabled                 bool                                             `json:"game_hall_enabled"`
	SupportTicketEnabled            bool                                             `json:"support_ticket_enabled"`
	GameSlotsEnabled                bool                                             `json:"game_slots_enabled"`
	GameSlotsMinBet                 float64                                          `json:"game_slots_min_bet"`
	GameSlotsMaxBet                 float64                                          `json:"game_slots_max_bet"`
	GameExchangeMinAmount           float64                                          `json:"game_exchange_min_amount"`
	GameExchangeMaxAmount           float64                                          `json:"game_exchange_max_amount"`
	GameExchangeDailyLimit          float64                                          `json:"game_exchange_daily_limit"`
	GameExchangeAllowDGToBalance    bool                                             `json:"game_exchange_allow_dg_to_balance"`
	CheckinEnabled                  bool                                             `json:"checkin_enabled"`
	CheckinLuckEnabled              bool                                             `json:"checkin_luck_enabled"`
	CheckinBlindboxEnabled          bool                                             `json:"checkin_blindbox_enabled"`
	RedPacketEnabled                bool                                             `json:"redpacket_enabled"`
	UsageQueryEnabled               bool                                             `json:"usage_query_enabled"`
	LeaderboardEnabled              bool                                             `json:"leaderboard_enabled"`
	LeaderboardBalanceEnabled       bool                                             `json:"leaderboard_balance_enabled"`
	LeaderboardConsumptionEnabled   bool                                             `json:"leaderboard_consumption_enabled"`
	LeaderboardCheckinEnabled       bool                                             `json:"leaderboard_checkin_enabled"`
	LeaderboardIncludeAdmin         bool                                             `json:"leaderboard_include_admin"`
	RechargePromotionEnabled        bool                                             `json:"recharge_promotion_enabled"`
	RechargePromotionMode           string                                           `json:"recharge_promotion_mode"`
	RechargePromotionPercentageRate float64                                          `json:"recharge_promotion_percentage_rate"`
	RechargePromotionMinimumAmount  float64                                          `json:"recharge_promotion_minimum_amount"`
	RechargePromotionMaximumBonus   float64                                          `json:"recharge_promotion_maximum_bonus"`
	RechargePromotionTierRules      []rechargepromotionsettings.TierRule             `json:"recharge_promotion_tier_rules"`
	RechargePromotionRateTiers      []rechargepromotionsettings.TieredPercentageRule `json:"recharge_promotion_tiered_percentage_rules"`
	RechargePromotionStartAt        string                                           `json:"recharge_promotion_start_at"`
	RechargePromotionEndAt          string                                           `json:"recharge_promotion_end_at"`
}

// Patch preserves the flat partial-update request contract. Applying it is the
// one place that maps those keys back to module-owned configuration.
type Patch struct {
	CodeFormatSettings                             *codeformatsettings.Config                        `json:"code_format_settings"`
	DefaultHomepage                                *string                                           `json:"default_homepage"`
	ThemeStyle                                     *string                                           `json:"theme_style"`
	EnableAppleTheme                               *bool                                             `json:"enable_apple_theme"`
	DefaultThemeMode                               *string                                           `json:"default_theme_mode"`
	GameHallEnabled                                *bool                                             `json:"game_hall_enabled"`
	RegistrationAliasProtectionEnabled             *bool                                             `json:"registration_alias_protection_enabled"`
	RegistrationRewardFingerprintProtectionEnabled *bool                                             `json:"registration_reward_fingerprint_protection_enabled"`
	RegistrationClusterProtectionEnabled           *bool                                             `json:"registration_cluster_protection_enabled"`
	RegistrationClusterWindow6h                    *int                                              `json:"registration_cluster_window_6h"`
	RegistrationClusterWindow24h                   *int                                              `json:"registration_cluster_window_24h"`
	RegistrationClusterMediumThreshold             *int                                              `json:"registration_cluster_medium_threshold"`
	RegistrationClusterHighThreshold               *int                                              `json:"registration_cluster_high_threshold"`
	RegistrationAutoHighRiskLockEnabled            *bool                                             `json:"registration_auto_high_risk_lock_enabled"`
	RegistrationTrialEnabled                       *bool                                             `json:"registration_trial_enabled"`
	RegistrationTrialHours                         *int                                              `json:"registration_trial_hours"`
	RegistrationTrialCostLimit                     *float64                                          `json:"registration_trial_cost_limit"`
	RegistrationTrialRPMLimit                      *int                                              `json:"registration_trial_rpm_limit"`
	RegistrationTrialConcurrency                   *int                                              `json:"registration_trial_concurrency"`
	RegistrationPaymentUnlockEnabled               *bool                                             `json:"registration_payment_unlock_enabled"`
	RegistrationPaymentUnlockAmount                *float64                                          `json:"registration_payment_unlock_amount"`
	SupportTicketEnabled                           *bool                                             `json:"support_ticket_enabled"`
	GameSlotsEnabled                               *bool                                             `json:"game_slots_enabled"`
	GameSlotsMinBet                                *float64                                          `json:"game_slots_min_bet"`
	GameSlotsMaxBet                                *float64                                          `json:"game_slots_max_bet"`
	GameExchangeMinAmount                          *float64                                          `json:"game_exchange_min_amount"`
	GameExchangeMaxAmount                          *float64                                          `json:"game_exchange_max_amount"`
	GameExchangeDailyLimit                         *float64                                          `json:"game_exchange_daily_limit"`
	GameExchangeAllowDGToBalance                   *bool                                             `json:"game_exchange_allow_dg_to_balance"`
	CheckinEnabled                                 *bool                                             `json:"checkin_enabled"`
	CheckinMinBalance                              *float64                                          `json:"checkin_min_balance"`
	CheckinMaxBalance                              *float64                                          `json:"checkin_max_balance"`
	CheckinLuckEnabled                             *bool                                             `json:"checkin_luck_enabled"`
	CheckinLuckMinMultiplier                       *float64                                          `json:"checkin_luck_min_multiplier"`
	CheckinLuckMaxMultiplier                       *float64                                          `json:"checkin_luck_max_multiplier"`
	CheckinBlindboxEnabled                         *bool                                             `json:"checkin_blindbox_enabled"`
	CheckinBlindboxTriggerType                     *string                                           `json:"checkin_blindbox_trigger_type"`
	CheckinBlindboxInterval                        *int                                              `json:"checkin_blindbox_interval"`
	RedPacketEnabled                               *bool                                             `json:"redpacket_enabled"`
	RedPacketMaxCount                              *int                                              `json:"redpacket_max_count"`
	RedPacketExpireHours                           *int                                              `json:"redpacket_expire_hours"`
	UsageQueryEnabled                              *bool                                             `json:"usage_query_enabled"`
	LeaderboardEnabled                             *bool                                             `json:"leaderboard_enabled"`
	LeaderboardBalanceEnabled                      *bool                                             `json:"leaderboard_balance_enabled"`
	LeaderboardConsumptionEnabled                  *bool                                             `json:"leaderboard_consumption_enabled"`
	LeaderboardCheckinEnabled                      *bool                                             `json:"leaderboard_checkin_enabled"`
	LeaderboardIncludeAdmin                        *bool                                             `json:"leaderboard_include_admin"`
	RechargePromotionEnabled                       *bool                                             `json:"recharge_promotion_enabled"`
	RechargePromotionMode                          *string                                           `json:"recharge_promotion_mode"`
	RechargePromotionPercentageRate                *float64                                          `json:"recharge_promotion_percentage_rate"`
	RechargePromotionMinimumAmount                 *float64                                          `json:"recharge_promotion_minimum_amount"`
	RechargePromotionMaximumBonus                  *float64                                          `json:"recharge_promotion_maximum_bonus"`
	RechargePromotionTierRules                     *[]rechargepromotionsettings.TierRule             `json:"recharge_promotion_tier_rules"`
	RechargePromotionRateTiers                     *[]rechargepromotionsettings.TieredPercentageRule `json:"recharge_promotion_tiered_percentage_rules"`
	RechargePromotionStartAt                       *string                                           `json:"recharge_promotion_start_at"`
	RechargePromotionEndAt                         *string                                           `json:"recharge_promotion_end_at"`
}

// HasChanges reports whether a request contains at least one Overlay-owned
// setting. It lets the admin settings handler avoid unrelated writes to the
// module store while keeping the flat JSON request contract.
func (p Patch) HasChanges() bool {
	return p.CodeFormatSettings != nil || p.DefaultHomepage != nil || p.ThemeStyle != nil || p.EnableAppleTheme != nil || p.DefaultThemeMode != nil || p.GameHallEnabled != nil || p.RegistrationAliasProtectionEnabled != nil || p.RegistrationRewardFingerprintProtectionEnabled != nil || p.RegistrationClusterProtectionEnabled != nil || p.RegistrationClusterWindow6h != nil || p.RegistrationClusterWindow24h != nil || p.RegistrationClusterMediumThreshold != nil || p.RegistrationClusterHighThreshold != nil || p.RegistrationAutoHighRiskLockEnabled != nil || p.RegistrationTrialEnabled != nil || p.RegistrationTrialHours != nil || p.RegistrationTrialCostLimit != nil || p.RegistrationTrialRPMLimit != nil || p.RegistrationTrialConcurrency != nil || p.RegistrationPaymentUnlockEnabled != nil || p.RegistrationPaymentUnlockAmount != nil || p.SupportTicketEnabled != nil || p.GameSlotsEnabled != nil || p.GameSlotsMinBet != nil || p.GameSlotsMaxBet != nil ||
		p.GameExchangeMinAmount != nil || p.GameExchangeMaxAmount != nil || p.GameExchangeDailyLimit != nil || p.GameExchangeAllowDGToBalance != nil ||
		p.CheckinEnabled != nil || p.CheckinMinBalance != nil || p.CheckinMaxBalance != nil || p.CheckinLuckEnabled != nil ||
		p.CheckinLuckMinMultiplier != nil || p.CheckinLuckMaxMultiplier != nil || p.CheckinBlindboxEnabled != nil ||
		p.CheckinBlindboxTriggerType != nil || p.CheckinBlindboxInterval != nil || p.RedPacketEnabled != nil ||
		p.RedPacketMaxCount != nil || p.RedPacketExpireHours != nil || p.UsageQueryEnabled != nil || p.LeaderboardEnabled != nil ||
		p.LeaderboardBalanceEnabled != nil || p.LeaderboardConsumptionEnabled != nil || p.LeaderboardCheckinEnabled != nil ||
		p.LeaderboardIncludeAdmin != nil || p.RechargePromotionEnabled != nil || p.RechargePromotionMode != nil ||
		p.RechargePromotionPercentageRate != nil || p.RechargePromotionMinimumAmount != nil || p.RechargePromotionMaximumBonus != nil ||
		p.RechargePromotionTierRules != nil || p.RechargePromotionRateTiers != nil || p.RechargePromotionStartAt != nil || p.RechargePromotionEndAt != nil
}

// Validate checks syntax that cannot be represented in a typed Config until
// after partial patches are applied. It keeps invalid time input from being
// silently ignored by the compatibility adapter.
func (p Patch) Validate() error {
	if p.ThemeStyle != nil && *p.ThemeStyle != applethemesettings.LegacyThemeGlass && *p.ThemeStyle != applethemesettings.LegacyThemeClassic {
		return fmt.Errorf("theme_style must be %q or %q", applethemesettings.LegacyThemeGlass, applethemesettings.LegacyThemeClassic)
	}
	if p.DefaultThemeMode != nil {
		if err := applethemesettings.Validate(applethemesettings.Config{Enabled: true, DefaultThemeMode: *p.DefaultThemeMode}); err != nil {
			return err
		}
	}
	if p.RechargePromotionStartAt != nil {
		if _, err := rechargepromotionsettings.ParseTime(*p.RechargePromotionStartAt); err != nil {
			return err
		}
	}
	if p.RechargePromotionEndAt != nil {
		if _, err := rechargepromotionsettings.ParseTime(*p.RechargePromotionEndAt); err != nil {
			return err
		}
	}
	return nil
}

func (r *Registry) Compatibility(ctx context.Context) (Compatibility, error) {
	snapshot, err := r.Read(ctx)
	if err != nil {
		return Compatibility{}, err
	}
	return CompatibilityFromSnapshot(snapshot), nil
}

func CompatibilityFromSnapshot(snapshot Snapshot) Compatibility {
	activity := snapshot.Activity
	appleTheme := snapshot.AppleTheme
	brandHome := snapshot.BrandHome
	codeFormat := snapshot.CodeFormat
	gameHall := snapshot.GameHall
	registrationProtection := snapshot.RegistrationProtection
	supportTickets := snapshot.SupportTickets
	rechargePromotions := snapshot.RechargePromotions
	return Compatibility{
		CodeFormatSettings: codeFormat,
		DefaultHomepage:    brandHome.DefaultHomepage,
		ThemeStyle:         legacyThemeStyle(appleTheme.Enabled),
		EnableAppleTheme:   appleTheme.Enabled,
		DefaultThemeMode:   appleTheme.DefaultThemeMode,
		GameHallEnabled:    gameHall.Enabled, GameSlotsEnabled: gameHall.SlotsEnabled,
		RegistrationAliasProtectionEnabled:             registrationProtection.AliasEnabled,
		RegistrationRewardFingerprintProtectionEnabled: registrationProtection.FingerprintEnabled,
		RegistrationClusterProtectionEnabled:           registrationProtection.ClusterEnabled,
		RegistrationClusterWindow6h:                    registrationProtection.ClusterWindow6h,
		RegistrationClusterWindow24h:                   registrationProtection.ClusterWindow24h,
		RegistrationClusterMediumThreshold:             registrationProtection.ClusterMedium,
		RegistrationClusterHighThreshold:               registrationProtection.ClusterHigh,
		RegistrationAutoHighRiskLockEnabled:            registrationProtection.AutoHighLock,
		RegistrationTrialEnabled:                       registrationProtection.TrialEnabled,
		RegistrationTrialHours:                         registrationProtection.TrialHours,
		RegistrationTrialCostLimit:                     registrationProtection.TrialCostLimit,
		RegistrationTrialRPMLimit:                      registrationProtection.TrialRPM,
		RegistrationTrialConcurrency:                   registrationProtection.TrialConcurrency,
		RegistrationPaymentUnlockEnabled:               registrationProtection.PaymentUnlock,
		RegistrationPaymentUnlockAmount:                registrationProtection.PaymentAmount,
		SupportTicketEnabled:                           supportTickets.Enabled,
		GameSlotsMinBet:                                gameHall.SlotsMinBet, GameSlotsMaxBet: gameHall.SlotsMaxBet,
		GameExchangeMinAmount: gameHall.ExchangeMinAmount, GameExchangeMaxAmount: gameHall.ExchangeMaxAmount,
		GameExchangeDailyLimit: gameHall.ExchangeDailyLimit, GameExchangeAllowDGToBalance: gameHall.ExchangeAllowDGToBalance,
		CheckinEnabled: activity.CheckinEnabled, CheckinMinBalance: activity.CheckinMinBalance, CheckinMaxBalance: activity.CheckinMaxBalance,
		CheckinLuckEnabled: activity.CheckinLuckEnabled, CheckinLuckMinMultiplier: activity.CheckinLuckMinMultiplier,
		CheckinLuckMaxMultiplier: activity.CheckinLuckMaxMultiplier, CheckinBlindboxEnabled: activity.CheckinBlindboxEnabled,
		CheckinBlindboxTriggerType: activity.CheckinBlindboxTriggerType, CheckinBlindboxInterval: activity.CheckinBlindboxInterval,
		RedPacketEnabled: activity.RedPacketEnabled, RedPacketMaxCount: activity.RedPacketMaxCount, RedPacketExpireHours: activity.RedPacketExpireHours,
		UsageQueryEnabled: activity.UsageQueryEnabled, LeaderboardEnabled: activity.LeaderboardEnabled,
		LeaderboardBalanceEnabled: activity.LeaderboardBalanceEnabled, LeaderboardConsumptionEnabled: activity.LeaderboardConsumptionEnabled,
		LeaderboardCheckinEnabled:       activity.LeaderboardCheckinEnabled,
		LeaderboardIncludeAdmin:         activity.LeaderboardIncludeAdmin,
		RechargePromotionEnabled:        rechargePromotions.Enabled,
		RechargePromotionMode:           rechargePromotions.Mode,
		RechargePromotionPercentageRate: rechargePromotions.PercentageRate,
		RechargePromotionMinimumAmount:  rechargePromotions.MinimumAmount,
		RechargePromotionMaximumBonus:   rechargePromotions.MaximumBonus,
		RechargePromotionTierRules:      rechargePromotions.TierRules,
		RechargePromotionRateTiers:      rechargePromotions.TieredPercentageRules,
		RechargePromotionStartAt:        rechargepromotionsettings.FormatTime(rechargePromotions.StartAt),
		RechargePromotionEndAt:          rechargepromotionsettings.FormatTime(rechargePromotions.EndAt),
	}
}

func (c Compatibility) Public() PublicCompatibility {
	return PublicCompatibility{
		DefaultHomepage:  c.DefaultHomepage,
		ThemeStyle:       c.ThemeStyle,
		EnableAppleTheme: c.EnableAppleTheme,
		DefaultThemeMode: c.DefaultThemeMode,
		GameHallEnabled:  c.GameHallEnabled, GameSlotsEnabled: c.GameSlotsEnabled,
		SupportTicketEnabled: c.SupportTicketEnabled,
		GameSlotsMinBet:      c.GameSlotsMinBet, GameSlotsMaxBet: c.GameSlotsMaxBet,
		GameExchangeMinAmount: c.GameExchangeMinAmount, GameExchangeMaxAmount: c.GameExchangeMaxAmount,
		GameExchangeDailyLimit: c.GameExchangeDailyLimit, GameExchangeAllowDGToBalance: c.GameExchangeAllowDGToBalance,
		CheckinEnabled: c.CheckinEnabled, CheckinLuckEnabled: c.CheckinLuckEnabled, CheckinBlindboxEnabled: c.CheckinBlindboxEnabled,
		RedPacketEnabled: c.RedPacketEnabled, UsageQueryEnabled: c.UsageQueryEnabled,
		LeaderboardEnabled: c.LeaderboardEnabled, LeaderboardBalanceEnabled: c.LeaderboardBalanceEnabled,
		LeaderboardConsumptionEnabled: c.LeaderboardConsumptionEnabled, LeaderboardCheckinEnabled: c.LeaderboardCheckinEnabled,
		LeaderboardIncludeAdmin:         c.LeaderboardIncludeAdmin,
		RechargePromotionEnabled:        c.RechargePromotionEnabled,
		RechargePromotionMode:           c.RechargePromotionMode,
		RechargePromotionPercentageRate: c.RechargePromotionPercentageRate,
		RechargePromotionMinimumAmount:  c.RechargePromotionMinimumAmount,
		RechargePromotionMaximumBonus:   c.RechargePromotionMaximumBonus,
		RechargePromotionTierRules:      c.RechargePromotionTierRules,
		RechargePromotionRateTiers:      c.RechargePromotionRateTiers,
		RechargePromotionStartAt:        c.RechargePromotionStartAt,
		RechargePromotionEndAt:          c.RechargePromotionEndAt,
	}
}

func (p Patch) Apply(snapshot Snapshot) Snapshot {
	activity := &snapshot.Activity
	appleTheme := &snapshot.AppleTheme
	brandHome := &snapshot.BrandHome
	codeFormat := &snapshot.CodeFormat
	gameHall := &snapshot.GameHall
	registrationProtection := &snapshot.RegistrationProtection
	supportTickets := &snapshot.SupportTickets
	rechargePromotions := &snapshot.RechargePromotions
	if p.CodeFormatSettings != nil {
		*codeFormat = *p.CodeFormatSettings
	}
	if p.DefaultHomepage != nil {
		brandHome.DefaultHomepage = *p.DefaultHomepage
	}
	if p.ThemeStyle != nil {
		appleTheme.Enabled = *p.ThemeStyle != applethemesettings.LegacyThemeClassic
	}
	if p.EnableAppleTheme != nil {
		appleTheme.Enabled = *p.EnableAppleTheme
	}
	if p.DefaultThemeMode != nil {
		appleTheme.DefaultThemeMode = *p.DefaultThemeMode
	}
	if p.GameHallEnabled != nil {
		gameHall.Enabled = *p.GameHallEnabled
	}
	if p.RegistrationAliasProtectionEnabled != nil {
		registrationProtection.AliasEnabled = *p.RegistrationAliasProtectionEnabled
	}
	if p.RegistrationRewardFingerprintProtectionEnabled != nil {
		registrationProtection.FingerprintEnabled = *p.RegistrationRewardFingerprintProtectionEnabled
	}
	if p.RegistrationClusterProtectionEnabled != nil {
		registrationProtection.ClusterEnabled = *p.RegistrationClusterProtectionEnabled
	}
	if p.RegistrationClusterWindow6h != nil {
		registrationProtection.ClusterWindow6h = *p.RegistrationClusterWindow6h
	}
	if p.RegistrationClusterWindow24h != nil {
		registrationProtection.ClusterWindow24h = *p.RegistrationClusterWindow24h
	}
	if p.RegistrationClusterMediumThreshold != nil {
		registrationProtection.ClusterMedium = *p.RegistrationClusterMediumThreshold
	}
	if p.RegistrationClusterHighThreshold != nil {
		registrationProtection.ClusterHigh = *p.RegistrationClusterHighThreshold
	}
	if p.RegistrationAutoHighRiskLockEnabled != nil {
		registrationProtection.AutoHighLock = *p.RegistrationAutoHighRiskLockEnabled
	}
	if p.RegistrationTrialEnabled != nil {
		registrationProtection.TrialEnabled = *p.RegistrationTrialEnabled
	}
	if p.RegistrationTrialHours != nil {
		registrationProtection.TrialHours = *p.RegistrationTrialHours
	}
	if p.RegistrationTrialCostLimit != nil {
		registrationProtection.TrialCostLimit = *p.RegistrationTrialCostLimit
	}
	if p.RegistrationTrialRPMLimit != nil {
		registrationProtection.TrialRPM = *p.RegistrationTrialRPMLimit
	}
	if p.RegistrationTrialConcurrency != nil {
		registrationProtection.TrialConcurrency = *p.RegistrationTrialConcurrency
	}
	if p.RegistrationPaymentUnlockEnabled != nil {
		registrationProtection.PaymentUnlock = *p.RegistrationPaymentUnlockEnabled
	}
	if p.RegistrationPaymentUnlockAmount != nil {
		registrationProtection.PaymentAmount = *p.RegistrationPaymentUnlockAmount
	}
	if p.SupportTicketEnabled != nil {
		supportTickets.Enabled = *p.SupportTicketEnabled
	}
	if p.GameSlotsEnabled != nil {
		gameHall.SlotsEnabled = *p.GameSlotsEnabled
	}
	if p.GameSlotsMinBet != nil {
		gameHall.SlotsMinBet = *p.GameSlotsMinBet
	}
	if p.GameSlotsMaxBet != nil {
		gameHall.SlotsMaxBet = *p.GameSlotsMaxBet
	}
	if p.GameExchangeMinAmount != nil {
		gameHall.ExchangeMinAmount = *p.GameExchangeMinAmount
	}
	if p.GameExchangeMaxAmount != nil {
		gameHall.ExchangeMaxAmount = *p.GameExchangeMaxAmount
	}
	if p.GameExchangeDailyLimit != nil {
		gameHall.ExchangeDailyLimit = *p.GameExchangeDailyLimit
	}
	if p.GameExchangeAllowDGToBalance != nil {
		gameHall.ExchangeAllowDGToBalance = *p.GameExchangeAllowDGToBalance
	}
	if p.CheckinEnabled != nil {
		activity.CheckinEnabled = *p.CheckinEnabled
	}
	if p.CheckinMinBalance != nil {
		activity.CheckinMinBalance = *p.CheckinMinBalance
	}
	if p.CheckinMaxBalance != nil {
		activity.CheckinMaxBalance = *p.CheckinMaxBalance
	}
	if p.CheckinLuckEnabled != nil {
		activity.CheckinLuckEnabled = *p.CheckinLuckEnabled
	}
	if p.CheckinLuckMinMultiplier != nil {
		activity.CheckinLuckMinMultiplier = *p.CheckinLuckMinMultiplier
	}
	if p.CheckinLuckMaxMultiplier != nil {
		activity.CheckinLuckMaxMultiplier = *p.CheckinLuckMaxMultiplier
	}
	if p.CheckinBlindboxEnabled != nil {
		activity.CheckinBlindboxEnabled = *p.CheckinBlindboxEnabled
	}
	if p.CheckinBlindboxTriggerType != nil {
		activity.CheckinBlindboxTriggerType = *p.CheckinBlindboxTriggerType
	}
	if p.CheckinBlindboxInterval != nil {
		activity.CheckinBlindboxInterval = *p.CheckinBlindboxInterval
	}
	if p.RedPacketEnabled != nil {
		activity.RedPacketEnabled = *p.RedPacketEnabled
	}
	if p.RedPacketMaxCount != nil {
		activity.RedPacketMaxCount = *p.RedPacketMaxCount
	}
	if p.RedPacketExpireHours != nil {
		activity.RedPacketExpireHours = *p.RedPacketExpireHours
	}
	if p.UsageQueryEnabled != nil {
		activity.UsageQueryEnabled = *p.UsageQueryEnabled
	}
	if p.LeaderboardEnabled != nil {
		activity.LeaderboardEnabled = *p.LeaderboardEnabled
	}
	if p.LeaderboardBalanceEnabled != nil {
		activity.LeaderboardBalanceEnabled = *p.LeaderboardBalanceEnabled
	}
	if p.LeaderboardConsumptionEnabled != nil {
		activity.LeaderboardConsumptionEnabled = *p.LeaderboardConsumptionEnabled
	}
	if p.LeaderboardCheckinEnabled != nil {
		activity.LeaderboardCheckinEnabled = *p.LeaderboardCheckinEnabled
	}
	if p.LeaderboardIncludeAdmin != nil {
		activity.LeaderboardIncludeAdmin = *p.LeaderboardIncludeAdmin
	}
	if p.RechargePromotionEnabled != nil {
		rechargePromotions.Enabled = *p.RechargePromotionEnabled
	}
	if p.RechargePromotionMode != nil {
		rechargePromotions.Mode = *p.RechargePromotionMode
	}
	if p.RechargePromotionPercentageRate != nil {
		rechargePromotions.PercentageRate = *p.RechargePromotionPercentageRate
	}
	if p.RechargePromotionMinimumAmount != nil {
		rechargePromotions.MinimumAmount = *p.RechargePromotionMinimumAmount
	}
	if p.RechargePromotionMaximumBonus != nil {
		rechargePromotions.MaximumBonus = *p.RechargePromotionMaximumBonus
	}
	if p.RechargePromotionTierRules != nil {
		rechargePromotions.TierRules = append([]rechargepromotionsettings.TierRule(nil), (*p.RechargePromotionTierRules)...)
	}
	if p.RechargePromotionRateTiers != nil {
		rechargePromotions.TieredPercentageRules = append([]rechargepromotionsettings.TieredPercentageRule(nil), (*p.RechargePromotionRateTiers)...)
	}
	if p.RechargePromotionStartAt != nil {
		parsed, err := rechargepromotionsettings.ParseTime(*p.RechargePromotionStartAt)
		if err == nil {
			rechargePromotions.StartAt = parsed
		}
	}
	if p.RechargePromotionEndAt != nil {
		parsed, err := rechargepromotionsettings.ParseTime(*p.RechargePromotionEndAt)
		if err == nil {
			rechargePromotions.EndAt = parsed
		}
	}
	return snapshot
}

func legacyThemeStyle(enabled bool) string {
	if enabled {
		return applethemesettings.LegacyThemeGlass
	}
	return applethemesettings.LegacyThemeClassic
}
