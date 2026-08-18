// Package registrationprotection implements the optional registration email
// alias policy without coupling the upstream authentication service to the
// Overlay settings store.
package registrationprotection

import (
	"context"
	"errors"
	"fmt"
	"strings"

	registrationsettings "github.com/Wei-Shaw/sub2api/internal/custom/modules/registration-protection/settings"
	"github.com/Wei-Shaw/sub2api/internal/custom/settings/contract"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

var gmailDomains = map[string]struct{}{
	"gmail.com":      {},
	"googlemail.com": {},
}

type Policy struct {
	store       contract.Store
	fingerprint *Store
	risk        *RiskStore
}

func NewPolicy(store contract.Store, fingerprint ...*Store) *Policy {
	var fp *Store
	if len(fingerprint) > 0 {
		fp = fingerprint[0]
	}
	var risk *RiskStore
	if fp != nil {
		risk = NewRiskStore(fp.db, store)
	}
	return &Policy{store: store, fingerprint: fp, risk: risk}
}

func (p *Policy) SetRedis(client *redis.Client) {
	if p != nil && p.risk != nil {
		p.risk.SetRedis(client)
	}
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
	if p == nil || (p.fingerprint == nil && p.risk == nil) {
		return nil, nil
	}
	if p.store == nil {
		return nil, fmt.Errorf("registration protection settings store is unavailable")
	}
	values, err := p.store.GetMultiple(ctx, settingsKeys())
	if err != nil {
		return nil, err
	}
	input := BeginInput{ClientIP: clientIP, UserAgent: userAgent, Source: signupSource}
	if !fingerprintEnabled(values) {
		if !contract.Bool(values, registrationsettings.KeyClusterEnabled, true) {
			return nil, nil
		}
		return riskReservation{risk: p.risk, input: input}, nil
	}
	reservation, err := p.fingerprint.Begin(ctx, input)
	if err != nil {
		return nil, err
	}
	return riskReservation{store: p.fingerprint, reservation: reservation, risk: p.risk, input: input}, nil
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
	if fingerprintEnabled(values) && p.fingerprint != nil {
		blocked, err := p.fingerprint.IsRewardBlocked(ctx, userID)
		if err != nil {
			return infraerrors.ServiceUnavailable("REGISTRATION_REWARD_PROTECTION_UNAVAILABLE", "registration reward protection is temporarily unavailable").WithCause(err)
		}
		if blocked {
			return infraerrors.Forbidden("CHECKIN_REWARD_NOT_ALLOWED", "check-in rewards are not available for this account")
		}
	}
	if p.risk != nil {
		_ = p.risk.RecordBehavior(ctx, userID, "fast_checkin")
		decision, err := p.risk.Decision(ctx, userID)
		if err != nil {
			return infraerrors.ServiceUnavailable("REGISTRATION_REWARD_PROTECTION_UNAVAILABLE", "registration reward protection is temporarily unavailable").WithCause(err)
		}
		if decision.Action == RiskActionMedium || decision.Action == RiskActionHigh {
			return infraerrors.Forbidden("CHECKIN_REWARD_NOT_ALLOWED", "check-in rewards are not available for this account")
		}
	}
	return nil
}

// AllowAPIKeyCreation is the narrow policy consumed by APIKeyService. It
// never affects login or existing keys.
func (p *Policy) AllowAPIKeyCreation(ctx context.Context, userID int64) error {
	if p == nil || p.risk == nil {
		return nil
	}
	_ = p.risk.RecordBehavior(ctx, userID, "api_key_created")
	decision, err := p.risk.Decision(ctx, userID)
	if err != nil {
		return infraerrors.ServiceUnavailable("REGISTRATION_RISK_UNAVAILABLE", "registration risk protection is temporarily unavailable").WithCause(err)
	}
	if decision.Action == RiskActionMedium || decision.Action == RiskActionHigh {
		return infraerrors.Forbidden("API_KEY_CREATION_RESTRICTED", "API key creation is temporarily restricted for this account")
	}
	return nil
}

func (p *Policy) AllowAPIRequest(ctx context.Context, userID int64) error {
	if p == nil || p.risk == nil {
		return nil
	}
	decision, err := p.risk.Decision(ctx, userID)
	if err != nil {
		return infraerrors.ServiceUnavailable("REGISTRATION_RISK_UNAVAILABLE", "registration risk protection is temporarily unavailable").WithCause(err)
	}
	if decision.Action == RiskActionHigh {
		return infraerrors.Forbidden("ACCESS_DENIED", "API access is temporarily restricted for this account")
	}
	if p.risk.TrialActive(ctx, userID) {
		if err := p.risk.AllowTrialCost(ctx, userID); err != nil {
			if errors.Is(err, ErrTrialCostLimit) {
				return infraerrors.Forbidden("REGISTRATION_TRIAL_COST_LIMIT", "new-user trial cost limit exceeded")
			}
			return infraerrors.ServiceUnavailable("REGISTRATION_RISK_UNAVAILABLE", "registration risk protection is temporarily unavailable").WithCause(err)
		}
		if err := p.risk.AllowTrialRPM(ctx, userID); err != nil {
			if errors.Is(err, ErrTrialRPMLimit) {
				return infraerrors.TooManyRequests("REGISTRATION_TRIAL_RPM_LIMIT", "new-user trial rate limit exceeded")
			}
			return infraerrors.ServiceUnavailable("REGISTRATION_RISK_UNAVAILABLE", "registration risk protection is temporarily unavailable").WithCause(err)
		}
	}
	return nil
}

func (p *Policy) EffectiveConcurrency(ctx context.Context, userID int64, current int) int {
	if p == nil || p.risk == nil || !p.risk.TrialActive(ctx, userID) {
		return current
	}
	config := registrationsettings.Default()
	if p.store != nil {
		if values, err := p.store.GetMultiple(ctx, registrationsettings.Keys()); err == nil {
			config = registrationsettings.FromValues(values)
		}
	}
	if current <= 0 || current > config.TrialConcurrency {
		return config.TrialConcurrency
	}
	return current
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
	risk        *RiskStore
	input       BeginInput
}

func (r riskReservation) RewardBlocked() bool {
	return r.reservation != nil && r.reservation.RewardBlocked()
}

func (r riskReservation) Commit(ctx context.Context, userID int64) error {
	exactRepeat := false
	if r.reservation != nil {
		result, err := r.store.Commit(ctx, r.reservation, userID)
		if err != nil {
			return err
		}
		exactRepeat = result.RewardBlocked
	}
	if r.risk == nil {
		return nil
	}
	_, err := r.risk.Evaluate(ctx, userID, r.input, exactRepeat)
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
