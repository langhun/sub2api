package checkin

import (
	"context"
	"fmt"

	dbent "github.com/Wei-Shaw/sub2api/ent"
)

// RedeemMetadataStore records activity-only wager metadata without extending
// the host application's redeem-code service model.
type RedeemMetadataStore interface {
	Store(context.Context, int64, float64, float64) error
}

// NewRedeemMetadataStore constructs the custom-owned metadata adapter.
func NewRedeemMetadataStore(client *dbent.Client) RedeemMetadataStore {
	return redeemMetadataStore{client: client}
}

type redeemMetadataStore struct{ client *dbent.Client }

func (s redeemMetadataStore) Store(ctx context.Context, redeemCodeID int64, multiplier, betAmount float64) error {
	if s.client == nil || redeemCodeID <= 0 || !validFinite(multiplier) || !validFinite(betAmount) {
		return fmt.Errorf("invalid redeem metadata")
	}
	_, err := entClient(ctx, s.client).ExecContext(ctx, `
		INSERT INTO custom_activity_redeem_metadata (
			redeem_code_id, multiplier, bet_amount, created_at, updated_at
		) VALUES ($1, $2, $3, NOW(), NOW())
		ON CONFLICT (redeem_code_id) DO UPDATE
		SET multiplier = EXCLUDED.multiplier,
			bet_amount = EXCLUDED.bet_amount,
			updated_at = NOW()`, redeemCodeID, multiplier, betAmount)
	return err
}

var _ RedeemMetadataStore = redeemMetadataStore{}
