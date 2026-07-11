package service

import (
	"context"
	"strconv"
	"strings"
)

type BalanceFeatureSettings struct {
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
	LeaderboardIncludeAdmin       bool
}

func parseBalanceFeatureSettings(values map[string]string) BalanceFeatureSettings {
	return BalanceFeatureSettings{
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
		LeaderboardIncludeAdmin:       values[SettingKeyLeaderboardIncludeAdmin] == "true",
	}
}

func appendBalanceFeatureUpdates(updates map[string]string, settings BalanceFeatureSettings) {
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
	updates[SettingKeyLeaderboardIncludeAdmin] = strconv.FormatBool(settings.LeaderboardIncludeAdmin)
}

func (s *SettingService) balanceFeatureSettings(ctx context.Context) BalanceFeatureSettings {
	values, err := s.settingRepo.GetMultiple(ctx, []string{
		SettingKeyCheckinEnabled, SettingKeyCheckinMinBalance, SettingKeyCheckinMaxBalance,
		SettingKeyCheckinLuckEnabled, SettingKeyCheckinLuckMinMultiplier, SettingKeyCheckinLuckMaxMultiplier,
		SettingKeyCheckinBlindboxEnabled, SettingKeyCheckinBlindboxTriggerType, SettingKeyCheckinBlindboxInterval,
		SettingKeyTransferEnabled, SettingKeyTransferFeeRate, SettingKeyTransferMinAmount, SettingKeyTransferMaxAmount,
		SettingKeyTransferDailyLimit, SettingKeyTransferDailyCountLimit, SettingKeyTransferVIPFeeExempt,
		SettingKeyRedPacketEnabled, SettingKeyRedPacketMaxCount, SettingKeyRedPacketExpireHours,
		SettingKeyUsageQueryEnabled, SettingKeyLeaderboardEnabled, SettingKeyLeaderboardBalanceEnabled,
		SettingKeyLeaderboardConsumptionEnabled, SettingKeyLeaderboardCheckinEnabled, SettingKeyLeaderboardIncludeAdmin,
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
	if err != nil || v < 0 {
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
