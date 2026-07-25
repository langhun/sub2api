package contract

import "context"

// Settings is the wallet-extension-owned subset of legacy balance-feature settings.
type Settings struct {
	DirectTransfer DirectTransferSettings
}

// DirectTransferSettings defines policy for point-to-point balance transfers.
type DirectTransferSettings struct {
	Enabled         bool
	FeeRate         float64
	MinimumAmount   float64
	MaximumAmount   float64
	DailyLimit      float64
	DailyCountLimit int
	VIPFeeExempt    bool
}

// SettingsReader reads effective wallet-extension settings for the current request.
type SettingsReader interface {
	GetWalletExtensionSettings(ctx context.Context) (Settings, error)
}
