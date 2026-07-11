package service

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
)

func (s BalanceFeatureSettings) ValidateGameHall() error {
	if math.IsNaN(s.GameSlotsMinBet) || math.IsInf(s.GameSlotsMinBet, 0) || s.GameSlotsMinBet <= 0 {
		return fmt.Errorf("game_slots_min_bet must be finite and greater than 0")
	}
	if math.IsNaN(s.GameSlotsMaxBet) || math.IsInf(s.GameSlotsMaxBet, 0) || s.GameSlotsMaxBet < s.GameSlotsMinBet {
		return fmt.Errorf("game_slots_max_bet must be finite and greater than or equal to game_slots_min_bet")
	}
	if math.IsNaN(s.GameExchangeMinAmount) || math.IsInf(s.GameExchangeMinAmount, 0) || s.GameExchangeMinAmount <= 0 {
		return fmt.Errorf("game_exchange_min_amount must be finite and greater than 0")
	}
	if !isFiniteNonnegative(s.GameExchangeMaxAmount) || (s.GameExchangeMaxAmount > 0 && s.GameExchangeMaxAmount < s.GameExchangeMinAmount) {
		return fmt.Errorf("game_exchange_max_amount must be 0 or greater than or equal to game_exchange_min_amount")
	}
	if !isFiniteNonnegative(s.GameExchangeDailyLimit) {
		return fmt.Errorf("game_exchange_daily_limit must be finite and nonnegative")
	}
	return nil
}

func (s BalanceFeatureSettings) Validate() error {
	if err := s.ValidateGameHall(); err != nil {
		return err
	}
	if err := validateFiniteRange("checkin balance", s.CheckinMinBalance, s.CheckinMaxBalance); err != nil {
		return err
	}
	if err := validateFiniteRange("checkin luck multiplier", s.CheckinLuckMinMultiplier, s.CheckinLuckMaxMultiplier); err != nil {
		return err
	}
	if !isFiniteNonnegative(s.TransferFeeRate) || s.TransferFeeRate > 1 {
		return fmt.Errorf("transfer_fee_rate must be finite and between 0 and 1")
	}
	for name, value := range map[string]float64{
		"transfer_min_amount":  s.TransferMinAmount,
		"transfer_max_amount":  s.TransferMaxAmount,
		"transfer_daily_limit": s.TransferDailyLimit,
	} {
		if !isFiniteNonnegative(value) {
			return fmt.Errorf("%s must be finite and nonnegative", name)
		}
	}
	if s.TransferMaxAmount > 0 && s.TransferMaxAmount < s.TransferMinAmount {
		return fmt.Errorf("transfer_max_amount must be 0 or greater than or equal to transfer_min_amount")
	}
	if s.CheckinBlindboxInterval < 0 || s.TransferDailyCountLimit < 0 || s.RedPacketMaxCount < 0 || s.RedPacketExpireHours < 0 {
		return fmt.Errorf("count, interval, and expiry settings must be nonnegative")
	}
	if s.CheckinBlindboxTriggerType != "" && s.CheckinBlindboxTriggerType != "streak" && s.CheckinBlindboxTriggerType != "total" {
		return fmt.Errorf("checkin_blindbox_trigger_type must be streak or total")
	}
	return nil
}

func validateFiniteRange(name string, minValue, maxValue float64) error {
	if !isFiniteNonnegative(minValue) || !isFiniteNonnegative(maxValue) {
		return fmt.Errorf("%s values must be finite and nonnegative", name)
	}
	if maxValue < minValue {
		return fmt.Errorf("%s maximum must be greater than or equal to minimum", name)
	}
	return nil
}

func isFiniteNonnegative(value float64) bool {
	return value >= 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
}

type BalanceFeatureSettings struct {
	GameHallEnabled               bool
	GameSlotsEnabled              bool
	GameSlotsMinBet               float64
	GameSlotsMaxBet               float64
	GameExchangeMinAmount         float64
	GameExchangeMaxAmount         float64
	GameExchangeDailyLimit        float64
	GameExchangeAllowDGToBalance  bool
	CheckinEnabled                bool
	CheckinMinBalance             float64
	CheckinMaxBalance             float64
	CheckinLuckEnabled            bool
	CheckinLuckMinMultiplier      float64
	CheckinLuckMaxMultiplier      float64
	CheckinBlindboxEnabled        bool
	CheckinBlindboxTriggerType    string
	CheckinBlindboxInterval       int
	TransferEnabled               bool
	TransferFeeRate               float64
	TransferMinAmount             float64
	TransferMaxAmount             float64
	TransferDailyLimit            float64
	TransferDailyCountLimit       int
	TransferVIPFeeExempt          bool
	RedPacketEnabled              bool
	RedPacketMaxCount             int
	RedPacketExpireHours          int
	UsageQueryEnabled             bool
	LeaderboardEnabled            bool
	LeaderboardBalanceEnabled     bool
	LeaderboardConsumptionEnabled bool
	LeaderboardCheckinEnabled     bool
	LeaderboardTransferEnabled    bool
	LeaderboardIncludeAdmin       bool
}

// GetMultiple exposes read-only setting access to focused feature services.
func (s *SettingService) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	return s.settingRepo.GetMultiple(ctx, keys)
}

func parseBalanceFeatureSettings(values map[string]string) BalanceFeatureSettings {
	return BalanceFeatureSettings{
		GameHallEnabled:               values[SettingKeyGameHallEnabled] == "true",
		GameSlotsEnabled:              values[SettingKeyGameSlotsEnabled] == "true",
		GameSlotsMinBet:               parseBalanceFeatureFloat(values[SettingKeyGameSlotsMinBet], 0.01),
		GameSlotsMaxBet:               parseBalanceFeatureFloat(values[SettingKeyGameSlotsMaxBet], 1000),
		GameExchangeMinAmount:         parseBalanceFeatureFloat(values[SettingKeyGameExchangeMinAmount], 0.01),
		GameExchangeMaxAmount:         parseBalanceFeatureFloat(values[SettingKeyGameExchangeMaxAmount], 1000),
		GameExchangeDailyLimit:        parseBalanceFeatureFloat(values[SettingKeyGameExchangeDailyLimit], 1000),
		GameExchangeAllowDGToBalance:  values[SettingKeyGameExchangeAllowDGToBalance] != "false",
		CheckinEnabled:                values[SettingKeyCheckinEnabled] == "true",
		CheckinMinBalance:             parseBalanceFeatureFloat(values[SettingKeyCheckinMinBalance], 0.1),
		CheckinMaxBalance:             parseBalanceFeatureFloat(values[SettingKeyCheckinMaxBalance], 1),
		CheckinLuckEnabled:            values[SettingKeyCheckinLuckEnabled] == "true",
		CheckinLuckMinMultiplier:      parseBalanceFeatureFloat(values[SettingKeyCheckinLuckMinMultiplier], 0.1),
		CheckinLuckMaxMultiplier:      parseBalanceFeatureFloat(values[SettingKeyCheckinLuckMaxMultiplier], 3),
		CheckinBlindboxEnabled:        values[SettingKeyCheckinBlindboxEnabled] == "true",
		CheckinBlindboxTriggerType:    parseBalanceFeatureChoice(values[SettingKeyCheckinBlindboxTriggerType], "streak", "streak", "total"),
		CheckinBlindboxInterval:       parseBalanceFeatureInt(values[SettingKeyCheckinBlindboxInterval], 7),
		TransferEnabled:               values[SettingKeyTransferEnabled] == "true",
		TransferFeeRate:               parseBalanceFeatureFloat(values[SettingKeyTransferFeeRate], 0.01),
		TransferMinAmount:             parseBalanceFeatureFloat(values[SettingKeyTransferMinAmount], 0.01),
		TransferMaxAmount:             parseBalanceFeatureFloat(values[SettingKeyTransferMaxAmount], 1000),
		TransferDailyLimit:            parseBalanceFeatureFloat(values[SettingKeyTransferDailyLimit], 1000),
		TransferDailyCountLimit:       parseBalanceFeatureInt(values[SettingKeyTransferDailyCountLimit], 50),
		TransferVIPFeeExempt:          values[SettingKeyTransferVIPFeeExempt] == "true",
		RedPacketEnabled:              values[SettingKeyRedPacketEnabled] == "true",
		RedPacketMaxCount:             parseBalanceFeatureInt(values[SettingKeyRedPacketMaxCount], 100),
		RedPacketExpireHours:          parseBalanceFeatureInt(values[SettingKeyRedPacketExpireHours], 24),
		UsageQueryEnabled:             values[SettingKeyUsageQueryEnabled] != "false",
		LeaderboardEnabled:            values[SettingKeyLeaderboardEnabled] != "false",
		LeaderboardBalanceEnabled:     values[SettingKeyLeaderboardBalanceEnabled] != "false",
		LeaderboardConsumptionEnabled: values[SettingKeyLeaderboardConsumptionEnabled] != "false",
		LeaderboardCheckinEnabled:     values[SettingKeyLeaderboardCheckinEnabled] != "false",
		LeaderboardTransferEnabled:    values[SettingKeyLeaderboardTransferEnabled] == "true",
		LeaderboardIncludeAdmin:       values[SettingKeyLeaderboardIncludeAdmin] == "true",
	}
}

func appendBalanceFeatureUpdates(updates map[string]string, settings BalanceFeatureSettings) {
	updates[SettingKeyGameHallEnabled] = strconv.FormatBool(settings.GameHallEnabled)
	updates[SettingKeyGameSlotsEnabled] = strconv.FormatBool(settings.GameSlotsEnabled)
	updates[SettingKeyGameSlotsMinBet] = strconv.FormatFloat(settings.GameSlotsMinBet, 'f', 8, 64)
	updates[SettingKeyGameSlotsMaxBet] = strconv.FormatFloat(settings.GameSlotsMaxBet, 'f', 8, 64)
	updates[SettingKeyGameExchangeMinAmount] = strconv.FormatFloat(settings.GameExchangeMinAmount, 'f', 8, 64)
	updates[SettingKeyGameExchangeMaxAmount] = strconv.FormatFloat(settings.GameExchangeMaxAmount, 'f', 8, 64)
	updates[SettingKeyGameExchangeDailyLimit] = strconv.FormatFloat(settings.GameExchangeDailyLimit, 'f', 8, 64)
	updates[SettingKeyGameExchangeAllowDGToBalance] = strconv.FormatBool(settings.GameExchangeAllowDGToBalance)
	updates[SettingKeyCheckinEnabled] = strconv.FormatBool(settings.CheckinEnabled)
	updates[SettingKeyCheckinMinBalance] = strconv.FormatFloat(settings.CheckinMinBalance, 'f', 8, 64)
	updates[SettingKeyCheckinMaxBalance] = strconv.FormatFloat(settings.CheckinMaxBalance, 'f', 8, 64)
	updates[SettingKeyCheckinLuckEnabled] = strconv.FormatBool(settings.CheckinLuckEnabled)
	updates[SettingKeyCheckinLuckMinMultiplier] = strconv.FormatFloat(settings.CheckinLuckMinMultiplier, 'f', 8, 64)
	updates[SettingKeyCheckinLuckMaxMultiplier] = strconv.FormatFloat(settings.CheckinLuckMaxMultiplier, 'f', 8, 64)
	updates[SettingKeyCheckinBlindboxEnabled] = strconv.FormatBool(settings.CheckinBlindboxEnabled)
	updates[SettingKeyCheckinBlindboxTriggerType] = parseBalanceFeatureChoice(settings.CheckinBlindboxTriggerType, "streak", "streak", "total")
	updates[SettingKeyCheckinBlindboxInterval] = strconv.Itoa(settings.CheckinBlindboxInterval)
	updates[SettingKeyTransferEnabled] = strconv.FormatBool(settings.TransferEnabled)
	updates[SettingKeyTransferFeeRate] = strconv.FormatFloat(settings.TransferFeeRate, 'f', 6, 64)
	updates[SettingKeyTransferMinAmount] = strconv.FormatFloat(settings.TransferMinAmount, 'f', 8, 64)
	updates[SettingKeyTransferMaxAmount] = strconv.FormatFloat(settings.TransferMaxAmount, 'f', 8, 64)
	updates[SettingKeyTransferDailyLimit] = strconv.FormatFloat(settings.TransferDailyLimit, 'f', 8, 64)
	updates[SettingKeyTransferDailyCountLimit] = strconv.Itoa(settings.TransferDailyCountLimit)
	updates[SettingKeyTransferVIPFeeExempt] = strconv.FormatBool(settings.TransferVIPFeeExempt)
	updates[SettingKeyRedPacketEnabled] = strconv.FormatBool(settings.RedPacketEnabled)
	updates[SettingKeyRedPacketMaxCount] = strconv.Itoa(settings.RedPacketMaxCount)
	updates[SettingKeyRedPacketExpireHours] = strconv.Itoa(settings.RedPacketExpireHours)
	updates[SettingKeyUsageQueryEnabled] = strconv.FormatBool(settings.UsageQueryEnabled)
	updates[SettingKeyLeaderboardEnabled] = strconv.FormatBool(settings.LeaderboardEnabled)
	updates[SettingKeyLeaderboardBalanceEnabled] = strconv.FormatBool(settings.LeaderboardBalanceEnabled)
	updates[SettingKeyLeaderboardConsumptionEnabled] = strconv.FormatBool(settings.LeaderboardConsumptionEnabled)
	updates[SettingKeyLeaderboardCheckinEnabled] = strconv.FormatBool(settings.LeaderboardCheckinEnabled)
	updates[SettingKeyLeaderboardTransferEnabled] = strconv.FormatBool(settings.LeaderboardTransferEnabled)
	updates[SettingKeyLeaderboardIncludeAdmin] = strconv.FormatBool(settings.LeaderboardIncludeAdmin)
}

func (s *SettingService) balanceFeatureSettings(ctx context.Context) BalanceFeatureSettings {
	values, err := s.settingRepo.GetMultiple(ctx, []string{
		SettingKeyGameHallEnabled, SettingKeyGameSlotsEnabled, SettingKeyGameSlotsMinBet, SettingKeyGameSlotsMaxBet,
		SettingKeyGameExchangeMinAmount, SettingKeyGameExchangeMaxAmount, SettingKeyGameExchangeDailyLimit, SettingKeyGameExchangeAllowDGToBalance,
		SettingKeyCheckinEnabled, SettingKeyCheckinMinBalance, SettingKeyCheckinMaxBalance,
		SettingKeyCheckinLuckEnabled, SettingKeyCheckinLuckMinMultiplier, SettingKeyCheckinLuckMaxMultiplier,
		SettingKeyCheckinBlindboxEnabled, SettingKeyCheckinBlindboxTriggerType, SettingKeyCheckinBlindboxInterval,
		SettingKeyTransferEnabled, SettingKeyTransferFeeRate, SettingKeyTransferMinAmount, SettingKeyTransferMaxAmount,
		SettingKeyTransferDailyLimit, SettingKeyTransferDailyCountLimit, SettingKeyTransferVIPFeeExempt,
		SettingKeyRedPacketEnabled, SettingKeyRedPacketMaxCount, SettingKeyRedPacketExpireHours,
		SettingKeyUsageQueryEnabled, SettingKeyLeaderboardEnabled, SettingKeyLeaderboardBalanceEnabled,
		SettingKeyLeaderboardConsumptionEnabled, SettingKeyLeaderboardCheckinEnabled, SettingKeyLeaderboardTransferEnabled, SettingKeyLeaderboardIncludeAdmin,
	})
	if err != nil {
		return parseBalanceFeatureSettings(nil)
	}
	return parseBalanceFeatureSettings(values)
}

func (s *SettingService) GetLeaderboardSettings(ctx context.Context) BalanceFeatureSettings {
	return s.balanceFeatureSettings(ctx)
}

func (s *SettingService) IsCheckinEnabled(ctx context.Context) bool {
	return s.balanceFeatureSettings(ctx).CheckinEnabled
}
func (s *SettingService) GetCheckinBalanceRange(ctx context.Context) (float64, float64) {
	v := s.balanceFeatureSettings(ctx)
	return v.CheckinMinBalance, v.CheckinMaxBalance
}
func (s *SettingService) IsCheckinLuckEnabled(ctx context.Context) bool {
	return s.balanceFeatureSettings(ctx).CheckinLuckEnabled
}
func (s *SettingService) GetCheckinLuckMultiplierRange(ctx context.Context) (float64, float64) {
	v := s.balanceFeatureSettings(ctx)
	return v.CheckinLuckMinMultiplier, v.CheckinLuckMaxMultiplier
}
func (s *SettingService) IsCheckinBlindboxEnabled(ctx context.Context) bool {
	return s.balanceFeatureSettings(ctx).CheckinBlindboxEnabled
}
func (s *SettingService) GetCheckinBlindboxTriggerType(ctx context.Context) string {
	return s.balanceFeatureSettings(ctx).CheckinBlindboxTriggerType
}
func (s *SettingService) GetCheckinBlindboxInterval(ctx context.Context) int {
	return s.balanceFeatureSettings(ctx).CheckinBlindboxInterval
}

func parseBalanceFeatureFloat(raw string, fallback float64) float64 {
	v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil || v < 0 || math.IsNaN(v) || math.IsInf(v, 0) {
		return fallback
	}
	return v
}
func parseBalanceFeatureInt(raw string, fallback int) int {
	v, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || v <= 0 {
		return fallback
	}
	return v
}
func parseBalanceFeatureChoice(raw, fallback string, allowed ...string) string {
	raw = strings.TrimSpace(raw)
	for _, v := range allowed {
		if raw == v {
			return raw
		}
	}
	return fallback
}
