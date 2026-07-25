package walletextension

import "github.com/Wei-Shaw/sub2api/internal/service"

// LegacyCompatibility isolates the remaining transfer operations that cannot
// move until activity owns red-packet-adjacent administration and rankings.
// It is a route adapter, not the implementation for direct user transfers.
type LegacyCompatibility struct {
	legacy *service.BalanceTransferService
}

// NewLegacyCompatibility exposes the legacy service only to compatibility handlers.
func NewLegacyCompatibility(legacy *service.BalanceTransferService) *LegacyCompatibility {
	return &LegacyCompatibility{legacy: legacy}
}
