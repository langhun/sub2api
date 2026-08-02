package supporttickets

import (
	"context"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/custom/settings/contract"
)

const (
	KeyEnabled = "support_ticket_enabled"
)

type Settings struct {
	Enabled bool
}
type SettingsReader interface {
	ReadSupportTickets(context.Context) (Settings, error)
}

func DefaultSettings() Settings { return Settings{Enabled: true} }
func SettingsKeys() []string    { return []string{KeyEnabled} }

func SettingsFromValues(values map[string]string) Settings {
	return Settings{Enabled: contract.Bool(values, KeyEnabled, true)}
}

func SettingsValues(config Settings) (map[string]string, error) {
	if err := ValidateSettings(config); err != nil {
		return nil, err
	}
	return map[string]string{KeyEnabled: fmt.Sprintf("%t", config.Enabled)}, nil
}

func ValidateSettings(Settings) error { return nil }
