// Package settings owns the persisted code-format configuration for custom
// features while preserving the established setting keys consumed by core
// redemption services.
package settings

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/custom/settings/contract"
)

const (
	KeyBalance      = "code_format_balance"
	KeyConcurrency  = "code_format_concurrency"
	KeySubscription = "code_format_subscription"
	KeyInvitation   = "code_format_invitation"
	KeyRedPacket    = "code_format_redpacket"
)

type Config struct {
	Balance      Format `json:"balance"`
	Concurrency  Format `json:"concurrency"`
	Subscription Format `json:"subscription"`
	Invitation   Format `json:"invitation"`
	RedPacket    Format `json:"redpacket"`
}

func Default() Config {
	return Config{
		Balance: DefaultCompactFormat(), Concurrency: DefaultCompactFormat(),
		Subscription: DefaultCompactFormat(), Invitation: DefaultCompactFormat(),
		RedPacket: DefaultRedPacketFormat(),
	}
}

func (c Config) Validate() error {
	for name, format := range map[string]Format{
		"balance": c.Balance, "concurrency": c.Concurrency, "subscription": c.Subscription,
		"invitation": c.Invitation, "redpacket": c.RedPacket,
	} {
		if err := format.Validate(); err != nil {
			return fmt.Errorf("%s code format: %w", name, err)
		}
	}
	return nil
}

func (c Config) RedeemFormat(codeType string) Format {
	switch codeType {
	case "concurrency", "admin_concurrency":
		return c.Concurrency
	case "subscription":
		return c.Subscription
	case "invitation":
		return c.Invitation
	default:
		return c.Balance
	}
}

type Reader struct{ store contract.Store }

func New(store contract.Store) *Reader { return &Reader{store: store} }

func (r *Reader) Read(ctx context.Context) (Config, error) {
	if r == nil || r.store == nil {
		return Config{}, fmt.Errorf("code format settings store is required")
	}
	values, err := r.store.GetMultiple(ctx, Keys())
	if err != nil {
		return Config{}, fmt.Errorf("read code format settings: %w", err)
	}
	return FromValues(values), nil
}

func Keys() []string {
	return []string{
		KeyBalance, KeyConcurrency, KeySubscription, KeyInvitation, KeyRedPacket,
	}
}

func FromValues(values map[string]string) Config {
	config := Default()
	decode(values[KeyBalance], &config.Balance)
	decode(values[KeyConcurrency], &config.Concurrency)
	decode(values[KeySubscription], &config.Subscription)
	decode(values[KeyInvitation], &config.Invitation)
	decode(values[KeyRedPacket], &config.RedPacket)
	return config
}

func Values(config Config) (map[string]string, error) {
	if config == (Config{}) {
		config = Default()
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}
	formats := map[string]Format{
		KeyBalance: config.Balance, KeyConcurrency: config.Concurrency, KeySubscription: config.Subscription,
		KeyInvitation: config.Invitation, KeyRedPacket: config.RedPacket,
	}
	values := make(map[string]string, len(formats))
	for key, format := range formats {
		encoded, err := json.Marshal(format)
		if err != nil {
			return nil, fmt.Errorf("encode %s: %w", key, err)
		}
		values[key] = string(encoded)
	}
	return values, nil
}

func Validate(config Config) error {
	return config.Validate()
}

func decode(raw string, target *Format) {
	if raw == "" || target == nil {
		return
	}
	var parsed Format
	if json.Unmarshal([]byte(raw), &parsed) != nil {
		return
	}
	if parsed.CharacterSet == characterSetLegacyHex {
		parsed.CharacterSet = CharacterSetAlphanumeric
	}
	if parsed.Validate() == nil {
		*target = parsed
	}
}
