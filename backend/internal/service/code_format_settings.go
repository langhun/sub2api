package service

import (
	"context"
	"encoding/json"
	"fmt"
)

const (
	SettingKeyCodeFormatBalance      = "code_format_balance"
	SettingKeyCodeFormatConcurrency  = "code_format_concurrency"
	SettingKeyCodeFormatSubscription = "code_format_subscription"
	SettingKeyCodeFormatInvitation   = "code_format_invitation"
	SettingKeyCodeFormatRedPacket    = "code_format_redpacket"
)

var codeFormatSettingKeys = []string{
	SettingKeyCodeFormatBalance,
	SettingKeyCodeFormatConcurrency,
	SettingKeyCodeFormatSubscription,
	SettingKeyCodeFormatInvitation,
	SettingKeyCodeFormatRedPacket,
}

type CodeFormatSettings struct {
	Balance      CodeFormatConfig `json:"balance"`
	Concurrency  CodeFormatConfig `json:"concurrency"`
	Subscription CodeFormatConfig `json:"subscription"`
	Invitation   CodeFormatConfig `json:"invitation"`
	RedPacket    CodeFormatConfig `json:"redpacket"`
}

func DefaultCodeFormatSettings() CodeFormatSettings {
	return CodeFormatSettings{
		Balance:      DefaultCompactRedeemCodeFormat(),
		Concurrency:  DefaultCompactRedeemCodeFormat(),
		Subscription: DefaultCompactRedeemCodeFormat(),
		Invitation:   DefaultCompactRedeemCodeFormat(),
		RedPacket:    DefaultRedPacketCodeFormat(),
	}
}

func (s CodeFormatSettings) Validate() error {
	formats := map[string]CodeFormatConfig{
		"balance":      s.Balance,
		"concurrency":  s.Concurrency,
		"subscription": s.Subscription,
		"invitation":   s.Invitation,
		"redpacket":    s.RedPacket,
	}
	for name, format := range formats {
		if err := format.Validate(); err != nil {
			return fmt.Errorf("%s code format: %w", name, err)
		}
	}
	return nil
}

func (s CodeFormatSettings) RedeemFormat(codeType string) CodeFormatConfig {
	switch codeType {
	case RedeemTypeConcurrency, AdjustmentTypeAdminConcurrency:
		return s.Concurrency
	case RedeemTypeSubscription:
		return s.Subscription
	case RedeemTypeInvitation:
		return s.Invitation
	default:
		return s.Balance
	}
}

// GenerateCode creates a code using the configured format for a redeem or
// internal adjustment type. A nil service falls back to the built-in defaults.
func (s *SettingService) GenerateCode(ctx context.Context, codeType string) (string, error) {
	settings := DefaultCodeFormatSettings()
	if s != nil {
		settings = s.GetCodeFormatSettings(ctx)
	}
	return settings.RedeemFormat(codeType).Generate()
}

func parseCodeFormatSettings(values map[string]string) CodeFormatSettings {
	settings := DefaultCodeFormatSettings()
	decodeCodeFormat(values[SettingKeyCodeFormatBalance], &settings.Balance)
	decodeCodeFormat(values[SettingKeyCodeFormatConcurrency], &settings.Concurrency)
	decodeCodeFormat(values[SettingKeyCodeFormatSubscription], &settings.Subscription)
	decodeCodeFormat(values[SettingKeyCodeFormatInvitation], &settings.Invitation)
	decodeCodeFormat(values[SettingKeyCodeFormatRedPacket], &settings.RedPacket)
	return settings
}

// ParseCodeFormatSettings exposes the existing storage compatibility rules to
// an Overlay settings owner without changing the code-generation behavior.
func ParseCodeFormatSettings(values map[string]string) CodeFormatSettings {
	return parseCodeFormatSettings(values)
}

func appendCodeFormatUpdates(updates map[string]string, settings CodeFormatSettings) error {
	if settings == (CodeFormatSettings{}) {
		settings = DefaultCodeFormatSettings()
	}
	if err := settings.Validate(); err != nil {
		return err
	}
	formats := map[string]CodeFormatConfig{
		SettingKeyCodeFormatBalance:      settings.Balance,
		SettingKeyCodeFormatConcurrency:  settings.Concurrency,
		SettingKeyCodeFormatSubscription: settings.Subscription,
		SettingKeyCodeFormatInvitation:   settings.Invitation,
		SettingKeyCodeFormatRedPacket:    settings.RedPacket,
	}
	for key, format := range formats {
		encoded, err := json.Marshal(format)
		if err != nil {
			return fmt.Errorf("encode %s: %w", key, err)
		}
		updates[key] = string(encoded)
	}
	return nil
}

// CodeFormatSettingsValues encodes the existing persisted format keys for an
// external settings owner. Redeem and adjustment code generation continue to
// read the same keys through GetCodeFormatSettings.
func CodeFormatSettingsValues(settings CodeFormatSettings) (map[string]string, error) {
	updates := make(map[string]string)
	if err := appendCodeFormatUpdates(updates, settings); err != nil {
		return nil, err
	}
	return updates, nil
}

func decodeCodeFormat(raw string, target *CodeFormatConfig) {
	if raw == "" || target == nil {
		return
	}
	var parsed CodeFormatConfig
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return
	}
	if parsed.CharacterSet == codeCharacterSetLegacyHex {
		parsed.CharacterSet = CodeCharacterSetAlphanumeric
	}
	if err := parsed.Validate(); err != nil {
		return
	}
	*target = parsed
}

func (s *SettingService) GetCodeFormatSettings(ctx context.Context) CodeFormatSettings {
	if s == nil || s.settingRepo == nil {
		return DefaultCodeFormatSettings()
	}
	values, err := s.settingRepo.GetMultiple(ctx, codeFormatSettingKeys)
	if err != nil {
		return DefaultCodeFormatSettings()
	}
	return parseCodeFormatSettings(values)
}
