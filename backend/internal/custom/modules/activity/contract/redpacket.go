package contract

import "context"

// RedPacketSettingsReader supplies the effective Activity-owned red-packet
// settings. It deliberately excludes transfer settings so the module never
// depends on BalanceTransferService for feature policy.
type RedPacketSettingsReader interface {
	GetActivityRedPacketSettings(ctx context.Context) (RedPacketSettings, error)
}
