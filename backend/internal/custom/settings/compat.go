package settings

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// Compatibility preserves the established flat admin settings contract while
// keeping the actual configuration split by Overlay module in Snapshot.
type Compatibility struct {
	CodeFormatSettings            service.CodeFormatSettings `json:"code_format_settings"`
	DefaultHomepage               string                     `json:"default_homepage"`
	GameHallEnabled               bool                       `json:"game_hall_enabled"`
	GameSlotsEnabled              bool                       `json:"game_slots_enabled"`
	GameSlotsMinBet               float64                    `json:"game_slots_min_bet"`
	GameSlotsMaxBet               float64                    `json:"game_slots_max_bet"`
	GameExchangeMinAmount         float64                    `json:"game_exchange_min_amount"`
	GameExchangeMaxAmount         float64                    `json:"game_exchange_max_amount"`
	GameExchangeDailyLimit        float64                    `json:"game_exchange_daily_limit"`
	GameExchangeAllowDGToBalance  bool                       `json:"game_exchange_allow_dg_to_balance"`
	CheckinEnabled                bool                       `json:"checkin_enabled"`
	CheckinMinBalance             float64                    `json:"checkin_min_balance"`
	CheckinMaxBalance             float64                    `json:"checkin_max_balance"`
	CheckinLuckEnabled            bool                       `json:"checkin_luck_enabled"`
	CheckinLuckMinMultiplier      float64                    `json:"checkin_luck_min_multiplier"`
	CheckinLuckMaxMultiplier      float64                    `json:"checkin_luck_max_multiplier"`
	CheckinBlindboxEnabled        bool                       `json:"checkin_blindbox_enabled"`
	CheckinBlindboxTriggerType    string                     `json:"checkin_blindbox_trigger_type"`
	CheckinBlindboxInterval       int                        `json:"checkin_blindbox_interval"`
	TransferEnabled               bool                       `json:"transfer_enabled"`
	TransferFeeRate               float64                    `json:"transfer_fee_rate"`
	TransferMinAmount             float64                    `json:"transfer_min_amount"`
	TransferMaxAmount             float64                    `json:"transfer_max_amount"`
	TransferDailyLimit            float64                    `json:"transfer_daily_limit"`
	TransferDailyCountLimit       int                        `json:"transfer_daily_count_limit"`
	TransferVIPFeeExempt          bool                       `json:"transfer_vip_fee_exempt"`
	RedPacketEnabled              bool                       `json:"redpacket_enabled"`
	RedPacketMaxCount             int                        `json:"redpacket_max_count"`
	RedPacketExpireHours          int                        `json:"redpacket_expire_hours"`
	UsageQueryEnabled             bool                       `json:"usage_query_enabled"`
	LeaderboardEnabled            bool                       `json:"leaderboard_enabled"`
	LeaderboardBalanceEnabled     bool                       `json:"leaderboard_balance_enabled"`
	LeaderboardConsumptionEnabled bool                       `json:"leaderboard_consumption_enabled"`
	LeaderboardCheckinEnabled     bool                       `json:"leaderboard_checkin_enabled"`
	LeaderboardTransferEnabled    bool                       `json:"leaderboard_transfer_enabled"`
	LeaderboardIncludeAdmin       bool                       `json:"leaderboard_include_admin"`
}

// PublicCompatibility is the intentionally smaller public projection. Values
// such as transfer limits and check-in amounts remain administrator-only.
type PublicCompatibility struct {
	DefaultHomepage               string  `json:"default_homepage"`
	GameHallEnabled               bool    `json:"game_hall_enabled"`
	GameSlotsEnabled              bool    `json:"game_slots_enabled"`
	GameSlotsMinBet               float64 `json:"game_slots_min_bet"`
	GameSlotsMaxBet               float64 `json:"game_slots_max_bet"`
	GameExchangeMinAmount         float64 `json:"game_exchange_min_amount"`
	GameExchangeMaxAmount         float64 `json:"game_exchange_max_amount"`
	GameExchangeDailyLimit        float64 `json:"game_exchange_daily_limit"`
	GameExchangeAllowDGToBalance  bool    `json:"game_exchange_allow_dg_to_balance"`
	CheckinEnabled                bool    `json:"checkin_enabled"`
	CheckinLuckEnabled            bool    `json:"checkin_luck_enabled"`
	CheckinBlindboxEnabled        bool    `json:"checkin_blindbox_enabled"`
	TransferEnabled               bool    `json:"transfer_enabled"`
	RedPacketEnabled              bool    `json:"redpacket_enabled"`
	UsageQueryEnabled             bool    `json:"usage_query_enabled"`
	LeaderboardEnabled            bool    `json:"leaderboard_enabled"`
	LeaderboardBalanceEnabled     bool    `json:"leaderboard_balance_enabled"`
	LeaderboardConsumptionEnabled bool    `json:"leaderboard_consumption_enabled"`
	LeaderboardCheckinEnabled     bool    `json:"leaderboard_checkin_enabled"`
	LeaderboardTransferEnabled    bool    `json:"leaderboard_transfer_enabled"`
	LeaderboardIncludeAdmin       bool    `json:"leaderboard_include_admin"`
}

// Patch preserves the flat partial-update request contract. Applying it is the
// one place that maps those keys back to module-owned configuration.
type Patch struct {
	CodeFormatSettings            *service.CodeFormatSettings `json:"code_format_settings"`
	DefaultHomepage               *string                     `json:"default_homepage"`
	GameHallEnabled               *bool                       `json:"game_hall_enabled"`
	GameSlotsEnabled              *bool                       `json:"game_slots_enabled"`
	GameSlotsMinBet               *float64                    `json:"game_slots_min_bet"`
	GameSlotsMaxBet               *float64                    `json:"game_slots_max_bet"`
	GameExchangeMinAmount         *float64                    `json:"game_exchange_min_amount"`
	GameExchangeMaxAmount         *float64                    `json:"game_exchange_max_amount"`
	GameExchangeDailyLimit        *float64                    `json:"game_exchange_daily_limit"`
	GameExchangeAllowDGToBalance  *bool                       `json:"game_exchange_allow_dg_to_balance"`
	CheckinEnabled                *bool                       `json:"checkin_enabled"`
	CheckinMinBalance             *float64                    `json:"checkin_min_balance"`
	CheckinMaxBalance             *float64                    `json:"checkin_max_balance"`
	CheckinLuckEnabled            *bool                       `json:"checkin_luck_enabled"`
	CheckinLuckMinMultiplier      *float64                    `json:"checkin_luck_min_multiplier"`
	CheckinLuckMaxMultiplier      *float64                    `json:"checkin_luck_max_multiplier"`
	CheckinBlindboxEnabled        *bool                       `json:"checkin_blindbox_enabled"`
	CheckinBlindboxTriggerType    *string                     `json:"checkin_blindbox_trigger_type"`
	CheckinBlindboxInterval       *int                        `json:"checkin_blindbox_interval"`
	TransferEnabled               *bool                       `json:"transfer_enabled"`
	TransferFeeRate               *float64                    `json:"transfer_fee_rate"`
	TransferMinAmount             *float64                    `json:"transfer_min_amount"`
	TransferMaxAmount             *float64                    `json:"transfer_max_amount"`
	TransferDailyLimit            *float64                    `json:"transfer_daily_limit"`
	TransferDailyCountLimit       *int                        `json:"transfer_daily_count_limit"`
	TransferVIPFeeExempt          *bool                       `json:"transfer_vip_fee_exempt"`
	RedPacketEnabled              *bool                       `json:"redpacket_enabled"`
	RedPacketMaxCount             *int                        `json:"redpacket_max_count"`
	RedPacketExpireHours          *int                        `json:"redpacket_expire_hours"`
	UsageQueryEnabled             *bool                       `json:"usage_query_enabled"`
	LeaderboardEnabled            *bool                       `json:"leaderboard_enabled"`
	LeaderboardBalanceEnabled     *bool                       `json:"leaderboard_balance_enabled"`
	LeaderboardConsumptionEnabled *bool                       `json:"leaderboard_consumption_enabled"`
	LeaderboardCheckinEnabled     *bool                       `json:"leaderboard_checkin_enabled"`
	LeaderboardTransferEnabled    *bool                       `json:"leaderboard_transfer_enabled"`
	LeaderboardIncludeAdmin       *bool                       `json:"leaderboard_include_admin"`
}

// HasChanges reports whether a request contains at least one Overlay-owned
// setting. It lets the admin settings handler avoid unrelated writes to the
// module store while keeping the flat JSON request contract.
func (p Patch) HasChanges() bool {
	return p.CodeFormatSettings != nil || p.DefaultHomepage != nil || p.GameHallEnabled != nil || p.GameSlotsEnabled != nil || p.GameSlotsMinBet != nil || p.GameSlotsMaxBet != nil ||
		p.GameExchangeMinAmount != nil || p.GameExchangeMaxAmount != nil || p.GameExchangeDailyLimit != nil || p.GameExchangeAllowDGToBalance != nil ||
		p.CheckinEnabled != nil || p.CheckinMinBalance != nil || p.CheckinMaxBalance != nil || p.CheckinLuckEnabled != nil ||
		p.CheckinLuckMinMultiplier != nil || p.CheckinLuckMaxMultiplier != nil || p.CheckinBlindboxEnabled != nil ||
		p.CheckinBlindboxTriggerType != nil || p.CheckinBlindboxInterval != nil || p.TransferEnabled != nil ||
		p.TransferFeeRate != nil || p.TransferMinAmount != nil || p.TransferMaxAmount != nil || p.TransferDailyLimit != nil ||
		p.TransferDailyCountLimit != nil || p.TransferVIPFeeExempt != nil || p.RedPacketEnabled != nil ||
		p.RedPacketMaxCount != nil || p.RedPacketExpireHours != nil || p.UsageQueryEnabled != nil || p.LeaderboardEnabled != nil ||
		p.LeaderboardBalanceEnabled != nil || p.LeaderboardConsumptionEnabled != nil || p.LeaderboardCheckinEnabled != nil ||
		p.LeaderboardTransferEnabled != nil || p.LeaderboardIncludeAdmin != nil
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
	brandHome := snapshot.BrandHome
	codeFormat := snapshot.CodeFormat
	wallet := snapshot.WalletExtension
	gameHall := snapshot.GameHall
	return Compatibility{
		CodeFormatSettings: codeFormat,
		DefaultHomepage:    brandHome.DefaultHomepage,
		GameHallEnabled:    gameHall.Enabled, GameSlotsEnabled: gameHall.SlotsEnabled,
		GameSlotsMinBet: gameHall.SlotsMinBet, GameSlotsMaxBet: gameHall.SlotsMaxBet,
		GameExchangeMinAmount: gameHall.ExchangeMinAmount, GameExchangeMaxAmount: gameHall.ExchangeMaxAmount,
		GameExchangeDailyLimit: gameHall.ExchangeDailyLimit, GameExchangeAllowDGToBalance: gameHall.ExchangeAllowDGToBalance,
		CheckinEnabled: activity.CheckinEnabled, CheckinMinBalance: activity.CheckinMinBalance, CheckinMaxBalance: activity.CheckinMaxBalance,
		CheckinLuckEnabled: activity.CheckinLuckEnabled, CheckinLuckMinMultiplier: activity.CheckinLuckMinMultiplier,
		CheckinLuckMaxMultiplier: activity.CheckinLuckMaxMultiplier, CheckinBlindboxEnabled: activity.CheckinBlindboxEnabled,
		CheckinBlindboxTriggerType: activity.CheckinBlindboxTriggerType, CheckinBlindboxInterval: activity.CheckinBlindboxInterval,
		TransferEnabled: wallet.DirectTransferEnabled, TransferFeeRate: wallet.DirectTransferFeeRate,
		TransferMinAmount: wallet.DirectTransferMinAmount, TransferMaxAmount: wallet.DirectTransferMaxAmount,
		TransferDailyLimit: wallet.DirectTransferDailyLimit, TransferDailyCountLimit: wallet.DirectTransferDailyCountLimit,
		TransferVIPFeeExempt: wallet.DirectTransferVIPFeeExempt,
		RedPacketEnabled:     activity.RedPacketEnabled, RedPacketMaxCount: activity.RedPacketMaxCount, RedPacketExpireHours: activity.RedPacketExpireHours,
		UsageQueryEnabled: activity.UsageQueryEnabled, LeaderboardEnabled: activity.LeaderboardEnabled,
		LeaderboardBalanceEnabled: activity.LeaderboardBalanceEnabled, LeaderboardConsumptionEnabled: activity.LeaderboardConsumptionEnabled,
		LeaderboardCheckinEnabled: activity.LeaderboardCheckinEnabled, LeaderboardTransferEnabled: activity.LeaderboardTransferEnabled,
		LeaderboardIncludeAdmin: activity.LeaderboardIncludeAdmin,
	}
}

func (c Compatibility) Public() PublicCompatibility {
	return PublicCompatibility{
		DefaultHomepage: c.DefaultHomepage,
		GameHallEnabled: c.GameHallEnabled, GameSlotsEnabled: c.GameSlotsEnabled,
		GameSlotsMinBet: c.GameSlotsMinBet, GameSlotsMaxBet: c.GameSlotsMaxBet,
		GameExchangeMinAmount: c.GameExchangeMinAmount, GameExchangeMaxAmount: c.GameExchangeMaxAmount,
		GameExchangeDailyLimit: c.GameExchangeDailyLimit, GameExchangeAllowDGToBalance: c.GameExchangeAllowDGToBalance,
		CheckinEnabled: c.CheckinEnabled, CheckinLuckEnabled: c.CheckinLuckEnabled, CheckinBlindboxEnabled: c.CheckinBlindboxEnabled,
		TransferEnabled: c.TransferEnabled, RedPacketEnabled: c.RedPacketEnabled, UsageQueryEnabled: c.UsageQueryEnabled,
		LeaderboardEnabled: c.LeaderboardEnabled, LeaderboardBalanceEnabled: c.LeaderboardBalanceEnabled,
		LeaderboardConsumptionEnabled: c.LeaderboardConsumptionEnabled, LeaderboardCheckinEnabled: c.LeaderboardCheckinEnabled,
		LeaderboardTransferEnabled: c.LeaderboardTransferEnabled, LeaderboardIncludeAdmin: c.LeaderboardIncludeAdmin,
	}
}

func (p Patch) Apply(snapshot Snapshot) Snapshot {
	activity := &snapshot.Activity
	brandHome := &snapshot.BrandHome
	codeFormat := &snapshot.CodeFormat
	wallet := &snapshot.WalletExtension
	gameHall := &snapshot.GameHall
	if p.CodeFormatSettings != nil {
		*codeFormat = *p.CodeFormatSettings
	}
	if p.DefaultHomepage != nil {
		brandHome.DefaultHomepage = *p.DefaultHomepage
	}
	if p.GameHallEnabled != nil {
		gameHall.Enabled = *p.GameHallEnabled
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
	if p.TransferEnabled != nil {
		wallet.DirectTransferEnabled = *p.TransferEnabled
	}
	if p.TransferFeeRate != nil {
		wallet.DirectTransferFeeRate = *p.TransferFeeRate
	}
	if p.TransferMinAmount != nil {
		wallet.DirectTransferMinAmount = *p.TransferMinAmount
	}
	if p.TransferMaxAmount != nil {
		wallet.DirectTransferMaxAmount = *p.TransferMaxAmount
	}
	if p.TransferDailyLimit != nil {
		wallet.DirectTransferDailyLimit = *p.TransferDailyLimit
	}
	if p.TransferDailyCountLimit != nil {
		wallet.DirectTransferDailyCountLimit = *p.TransferDailyCountLimit
	}
	if p.TransferVIPFeeExempt != nil {
		wallet.DirectTransferVIPFeeExempt = *p.TransferVIPFeeExempt
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
	if p.LeaderboardTransferEnabled != nil {
		activity.LeaderboardTransferEnabled = *p.LeaderboardTransferEnabled
	}
	if p.LeaderboardIncludeAdmin != nil {
		activity.LeaderboardIncludeAdmin = *p.LeaderboardIncludeAdmin
	}
	return snapshot
}
