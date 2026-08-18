package registrationprotection

import (
	"context"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// ServicePolicyAdapter exposes the fingerprint Store through the narrow
// upstream authentication hook. It keeps all persistence details inside the
// Overlay module while allowing AuthService to remain storage-agnostic.
type ServicePolicyAdapter struct {
	store *Store
}

func NewServicePolicyAdapter(store *Store) *ServicePolicyAdapter {
	return &ServicePolicyAdapter{store: store}
}

func (p *ServicePolicyAdapter) Begin(ctx context.Context, clientIP, userAgent, signupSource string) (service.RegistrationRiskReservation, error) {
	if p == nil || p.store == nil {
		return nil, ErrUnavailable
	}
	reservation, err := p.store.Begin(ctx, BeginInput{
		ClientIP:  clientIP,
		UserAgent: userAgent,
		Source:    signupSource,
	})
	if err != nil {
		return nil, err
	}
	return &serviceReservation{store: p.store, reservation: reservation}, nil
}

func (p *ServicePolicyAdapter) IsRewardBlocked(ctx context.Context, userID int64) (bool, error) {
	if p == nil || p.store == nil {
		return false, ErrUnavailable
	}
	return p.store.IsRewardBlocked(ctx, userID)
}

type serviceReservation struct {
	store       *Store
	reservation *Reservation
}

func (r *serviceReservation) RewardBlocked() bool {
	return r != nil && r.reservation != nil && r.reservation.RewardBlocked()
}

func (r *serviceReservation) Commit(ctx context.Context, userID int64) error {
	if r == nil || r.store == nil || r.reservation == nil {
		return ErrInvalidReservation
	}
	result, err := r.store.Commit(ctx, r.reservation, userID)
	if err != nil {
		return err
	}
	// AuthService reads RewardBlocked before creating a user. Serializable
	// reservations make a post-creation decision change unreachable in normal
	// operation; retain the final result for diagnostics and idempotent reads.
	if result.RewardBlocked {
		return nil
	}
	return nil
}

func (r *serviceReservation) Abort(ctx context.Context) error {
	if r == nil || r.store == nil || r.reservation == nil {
		return ErrInvalidReservation
	}
	if err := r.store.Abort(ctx, r.reservation); err != nil {
		return fmt.Errorf("abort registration risk reservation: %w", err)
	}
	return nil
}

var _ service.RegistrationRiskPolicy = (*ServicePolicyAdapter)(nil)
var _ service.RegistrationRiskReservation = (*serviceReservation)(nil)
