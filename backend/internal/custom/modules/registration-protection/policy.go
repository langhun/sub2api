// Package registrationprotection implements the optional registration email
// alias policy without coupling the upstream authentication service to the
// Overlay settings store.
package registrationprotection

import (
	"context"
	"fmt"
	"strings"

	registrationsettings "github.com/Wei-Shaw/sub2api/internal/custom/modules/registration-protection/settings"
	"github.com/Wei-Shaw/sub2api/internal/custom/settings/contract"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

var gmailDomains = map[string]struct{}{
	"gmail.com":      {},
	"googlemail.com": {},
}

type Policy struct {
	store        contract.Store
	fingerprint  *Store
}

func NewPolicy(store contract.Store, fingerprint ...*Store) *Policy {
	var fp *Store
	if len(fingerprint) > 0 {
		fp = fingerprint[0]
	}
	return &Policy{store: store, fingerprint: fp}
}

func (p *Policy) Validate(ctx context.Context, email string) error {
	if p == nil || p.store == nil {
		return nil
	}
	values, err := p.store.GetMultiple(ctx, settingsKeys())
	if err != nil {
		return fmt.Errorf("read registration protection settings: %w", err)
	}
	if !aliasEnabled(values) {
		return nil
	}
	local, domain, ok := splitEmail(email)
	if !ok {
		return nil
	}
	if strings.Contains(local, "+") {
		return service.ErrEmailAliasNotAllowed
	}
	if _, ok := gmailDomains[domain]; ok && strings.Contains(local, ".") {
		return service.ErrEmailAliasNotAllowed
	}
	return nil
}

func settingsKeys() []string { return registrationsettings.Keys() }

func aliasEnabled(values map[string]string) bool {
	return contract.Bool(values, registrationsettings.KeyAliasEnabled, true)
}

func fingerprintEnabled(values map[string]string) bool {
	return contract.Bool(values, registrationsettings.KeyFingerprintEnabled, true)
}

// Begin implements service.RegistrationRiskPolicy. When the feature is
// disabled, registration remains unchanged and no fingerprint is persisted.
func (p *Policy) Begin(ctx context.Context, clientIP, userAgent, signupSource string) (service.RegistrationRiskReservation, error) {
	if p == nil || p.fingerprint == nil {
		return nil, nil
	}
	if p.store == nil {
		return nil, fmt.Errorf("registration protection settings store is unavailable")
	}
	values, err := p.store.GetMultiple(ctx, settingsKeys())
	if err != nil {
		return nil, err
	}
	if !fingerprintEnabled(values) {
		return nil, nil
	}
	reservation, err := p.fingerprint.Begin(ctx, BeginInput{ClientIP: clientIP, UserAgent: userAgent, Source: signupSource})
	if err != nil {
		return nil, err
	}
	return riskReservation{store: p.fingerprint, reservation: reservation}, nil
}

// CheckinRewardAllowed implements the check-in reward gate. Existing accounts
// remain usable; only the reward operation is denied for repeated fingerprints.
func (p *Policy) CheckinRewardAllowed(ctx context.Context, userID int64) error {
	if p == nil || p.fingerprint == nil {
		return nil
	}
	if p.store == nil {
		return infraerrors.ServiceUnavailable("REGISTRATION_REWARD_PROTECTION_UNAVAILABLE", "registration reward protection is temporarily unavailable")
	}
	values, err := p.store.GetMultiple(ctx, settingsKeys())
	if err != nil {
		return infraerrors.ServiceUnavailable("REGISTRATION_REWARD_PROTECTION_UNAVAILABLE", "registration reward protection is temporarily unavailable").WithCause(err)
	}
	if !fingerprintEnabled(values) {
		return nil
	}
	blocked, err := p.fingerprint.IsRewardBlocked(ctx, userID)
	if err != nil {
		return infraerrors.ServiceUnavailable("REGISTRATION_REWARD_PROTECTION_UNAVAILABLE", "registration reward protection is temporarily unavailable").WithCause(err)
	}
	if blocked {
		return infraerrors.Forbidden("CHECKIN_REWARD_NOT_ALLOWED", "check-in rewards are not available for this account")
	}
	return nil
}

func (p *Policy) IsRewardBlocked(ctx context.Context, userID int64) (bool, error) {
	if p == nil || p.fingerprint == nil {
		return false, nil
	}
	if p.store == nil {
		return false, fmt.Errorf("registration protection settings store is unavailable")
	}
	values, err := p.store.GetMultiple(ctx, settingsKeys())
	if err != nil {
		return false, err
	}
	if !fingerprintEnabled(values) {
		return false, nil
	}
	return p.fingerprint.IsRewardBlocked(ctx, userID)
}

type riskReservation struct {
	store       *Store
	reservation *Reservation
}

func (r riskReservation) RewardBlocked() bool {
	return r.reservation != nil && r.reservation.RewardBlocked()
}

func (r riskReservation) Commit(ctx context.Context, userID int64) error {
	if r.reservation == nil {
		return nil
	}
	_, err := r.store.Commit(ctx, r.reservation, userID)
	return err
}

func (r riskReservation) Abort(ctx context.Context) error {
	if r.reservation == nil {
		return nil
	}
	return r.store.Abort(ctx, r.reservation)
}

func splitEmail(email string) (local, domain string, ok bool) {
	email = strings.ToLower(strings.TrimSpace(email))
	local, domain, ok = strings.Cut(email, "@")
	if !ok || local == "" || domain == "" || strings.Contains(domain, "@") {
		return "", "", false
	}
	domain = strings.TrimRight(domain, ".")
	if domain == "" {
		return "", "", false
	}
	return local, domain, true
}
