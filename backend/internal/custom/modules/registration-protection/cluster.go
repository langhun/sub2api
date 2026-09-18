package registrationprotection

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	registrationsettings "github.com/Wei-Shaw/sub2api/internal/custom/modules/registration-protection/settings"
	"github.com/Wei-Shaw/sub2api/internal/custom/settings/contract"
	"github.com/redis/go-redis/v9"
)

const (
	RiskActionNone   = "none"
	RiskActionMedium = "medium"
	RiskActionHigh   = "high"
)

var ErrTrialRPMLimit = fmt.Errorf("registration trial rpm limit exceeded")
var ErrTrialCostLimit = fmt.Errorf("registration trial cost limit exceeded")

type RiskDecision struct {
	Score       int
	Action      string
	Signals     []string
	ExpiresAt   time.Time
	NetworkHash string
	UAHash      string
	Network     string
}

// Preflight evaluates the signals that are already durable before an account
// exists. The current registration is included in the window counts so a
// high-risk attempt can be rejected before any balance or account side effect.
func (s *RiskStore) Preflight(ctx context.Context, input BeginInput, exactRepeat bool) (RiskDecision, error) {
	if s == nil || s.db == nil {
		return RiskDecision{}, ErrUnavailable
	}
	config := registrationsettings.Default()
	if s.settings != nil {
		values, err := s.settings.GetMultiple(ctx, registrationsettings.Keys())
		if err != nil {
			return RiskDecision{}, fmt.Errorf("read registration risk settings: %w", err)
		}
		config = registrationsettings.FromValues(values)
	}
	if !config.ClusterEnabled {
		return RiskDecision{Action: RiskActionNone}, nil
	}
	network, ok := normalizeNetwork(input.ClientIP)
	ua := NormalizeUserAgent(input.UserAgent)
	if !ok || ua == "" {
		return RiskDecision{Action: RiskActionNone}, nil
	}
	networkHash := hashValue(network)
	uaHash := hashValue(ua)
	var count6h, count24h int
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FILTER (WHERE registered_at >= NOW() - INTERVAL '6 hours'),
		       COUNT(*) FILTER (WHERE registered_at >= NOW() - INTERVAL '24 hours')
		FROM custom_registration_cluster_events
		WHERE network_hash = $1 AND user_agent_hash = $2`, networkHash, uaHash).Scan(&count6h, &count24h); err != nil {
		return RiskDecision{}, fmt.Errorf("count registration cluster events: %w", err)
	}
	return scoreRiskDecision(config, networkHash, uaHash, network, count6h+1, count24h+1, exactRepeat), nil
}

func scoreRiskDecision(config registrationsettings.Config, networkHash, uaHash, network string, count6h, count24h int, exactRepeat bool) RiskDecision {
	score := 0
	signals := make([]string, 0, 4)
	if count6h >= config.ClusterWindow6h {
		score += 3
		signals = append(signals, "cluster_6h")
	}
	if count24h >= config.ClusterWindow24h {
		score += 3
		signals = append(signals, "cluster_24h")
	}
	if exactRepeat {
		score += 2
		signals = append(signals, "exact_fingerprint_repeat")
	}
	action := RiskActionNone
	// Network/UA cluster counts are intentionally observation-only by
	// themselves. Shared NATs, offices, schools, and mobile carriers can
	// legitimately produce many registrations with the same normalized UA;
	// require an exact fingerprint repeat before taking the irreversible
	// high-risk registration path.
	if exactRepeat && score >= config.ClusterHigh {
		action = RiskActionHigh
	} else if score >= config.ClusterMedium {
		action = RiskActionMedium
	}
	if action == RiskActionHigh && !config.AutoHighLock {
		action = RiskActionMedium
	}
	return RiskDecision{Score: score, Action: action, Signals: signals, NetworkHash: networkHash, UAHash: uaHash, Network: network}
}

// RiskStore keeps cluster aggregation separate from the exact fingerprint
// tables, allowing older installations to migrate without changing their
// reward-history queries.
type RiskStore struct {
	db       *sql.DB
	settings contract.Store
	redis    *redis.Client
}

func NewRiskStore(db *sql.DB, settings contract.Store) *RiskStore {
	return &RiskStore{db: db, settings: settings}
}

func (s *RiskStore) SetRedis(client *redis.Client) {
	if s != nil {
		s.redis = client
	}
}

func (s *RiskStore) Evaluate(ctx context.Context, userID int64, input BeginInput, exactRepeat bool) (RiskDecision, error) {
	if s == nil || s.db == nil || userID <= 0 {
		return RiskDecision{}, ErrUnavailable
	}
	config := registrationsettings.Default()
	if s.settings != nil {
		values, err := s.settings.GetMultiple(ctx, registrationsettings.Keys())
		if err != nil {
			return RiskDecision{}, fmt.Errorf("read registration risk settings: %w", err)
		}
		config = registrationsettings.FromValues(values)
	}
	if !config.ClusterEnabled {
		return RiskDecision{Action: RiskActionNone}, nil
	}
	network, ok := normalizeNetwork(input.ClientIP)
	ua := NormalizeUserAgent(input.UserAgent)
	if !ok || ua == "" {
		return RiskDecision{Action: RiskActionNone}, nil
	}
	networkHash := hashValue(network)
	uaHash := hashValue(ua)
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO custom_registration_cluster_events (user_id, network_hash, user_agent_hash, network_prefix)
		VALUES ($1, $2, $3, $4)`, userID, networkHash, uaHash, network); err != nil {
		return RiskDecision{}, fmt.Errorf("record registration cluster event: %w", err)
	}
	var count6h, count24h int
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FILTER (WHERE registered_at >= NOW() - INTERVAL '6 hours'),
		       COUNT(*) FILTER (WHERE registered_at >= NOW() - INTERVAL '24 hours')
		FROM custom_registration_cluster_events
		WHERE network_hash = $1 AND user_agent_hash = $2`, networkHash, uaHash).Scan(&count6h, &count24h); err != nil {
		return RiskDecision{}, fmt.Errorf("count registration cluster events: %w", err)
	}
	decision := scoreRiskDecision(config, networkHash, uaHash, network, count6h, count24h, exactRepeat)
	score, signals, action := decision.Score, decision.Signals, decision.Action
	expires := time.Now().UTC().Add(24 * time.Hour)
	encoded, _ := json.Marshal(signals)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO custom_registration_risk_profiles (user_id, risk_score, risk_action, risk_signals, risk_expires_at, updated_at)
		VALUES ($1, $2, $3, $4::jsonb, $5, NOW())
		ON CONFLICT (user_id) DO UPDATE SET risk_score = EXCLUDED.risk_score,
		  risk_action = EXCLUDED.risk_action, risk_signals = EXCLUDED.risk_signals,
		  risk_expires_at = EXCLUDED.risk_expires_at, updated_at = NOW()
		WHERE custom_registration_risk_profiles.admin_override = ''`, userID, score, action, string(encoded), expires)
	if err != nil {
		return RiskDecision{}, fmt.Errorf("persist registration risk profile: %w", err)
	}
	decision.ExpiresAt = expires
	return decision, nil
}

func (s *RiskStore) Decision(ctx context.Context, userID int64) (RiskDecision, error) {
	if s == nil || s.db == nil {
		return RiskDecision{}, ErrUnavailable
	}
	var d RiskDecision
	var signals []byte
	var expires *time.Time
	var paymentUnlocked bool
	err := s.db.QueryRowContext(ctx, `SELECT risk_score, risk_action, risk_signals, risk_expires_at, payment_unlocked FROM custom_registration_risk_profiles WHERE user_id = $1`, userID).Scan(&d.Score, &d.Action, &signals, &expires, &paymentUnlocked)
	if err == sql.ErrNoRows {
		return RiskDecision{Action: RiskActionNone}, nil
	}
	if err != nil {
		return RiskDecision{}, err
	}
	if d.Action == RiskActionNone && !paymentUnlocked {
		_ = json.Unmarshal(signals, &d.Signals)
		if expires != nil {
			d.ExpiresAt = *expires
		}
		return d, nil
	}
	config := registrationsettings.Default()
	if s.settings != nil {
		values, settingsErr := s.settings.GetMultiple(ctx, registrationsettings.Keys())
		if settingsErr != nil {
			return RiskDecision{}, settingsErr
		}
		config = registrationsettings.FromValues(values)
	}
	if config.PaymentUnlock {
		var paid float64
		if paymentErr := s.db.QueryRowContext(ctx, `
			SELECT COALESCE(SUM(pay_amount - refund_amount), 0)
			FROM payment_orders
			WHERE user_id = $1 AND status = 'COMPLETED' AND pay_amount > refund_amount`, userID).Scan(&paid); paymentErr == nil && paid >= config.PaymentAmount {
			_, _ = s.db.ExecContext(ctx, `UPDATE custom_registration_risk_profiles SET risk_action = 'none', payment_unlocked = TRUE, updated_at = NOW() WHERE user_id = $1 AND admin_override = ''`, userID)
			return RiskDecision{Action: RiskActionNone}, nil
		}
		if paymentUnlocked && d.Action != "manual_lock" {
			if d.Score >= config.ClusterHigh {
				d.Action = RiskActionHigh
				if !config.AutoHighLock {
					d.Action = RiskActionMedium
				}
			} else if d.Score >= config.ClusterMedium {
				d.Action = RiskActionMedium
			} else {
				d.Action = RiskActionNone
			}
			_, _ = s.db.ExecContext(ctx, `UPDATE custom_registration_risk_profiles SET payment_unlocked = FALSE, risk_action = $2, updated_at = NOW() WHERE user_id = $1 AND admin_override = ''`, userID, d.Action)
		}
	}
	if expires != nil && expires.Before(time.Now().UTC()) && d.Action != "manual_lock" {
		return RiskDecision{Action: RiskActionNone}, nil
	}
	_ = json.Unmarshal(signals, &d.Signals)
	if expires != nil {
		d.ExpiresAt = *expires
	}
	return d, nil
}

// RecordBehavior adds a bounded, explainable signal after registration. It
// only scores events inside the first 90 seconds and keeps the raw request out
// of storage.
func (s *RiskStore) RecordBehavior(ctx context.Context, userID int64, signal string) error {
	if s == nil || s.db == nil || userID <= 0 || strings.TrimSpace(signal) == "" {
		return nil
	}
	var recent bool
	if err := s.db.QueryRowContext(ctx, `
		SELECT EXISTS (SELECT 1 FROM custom_registration_cluster_events WHERE user_id = $1 AND registered_at >= NOW() - INTERVAL '90 seconds')`, userID).Scan(&recent); err != nil {
		return err
	}
	if !recent {
		return nil
	}
	config := registrationsettings.Default()
	if s.settings != nil {
		values, err := s.settings.GetMultiple(ctx, registrationsettings.Keys())
		if err != nil {
			return err
		}
		config = registrationsettings.FromValues(values)
	}
	highAction := RiskActionHigh
	if !config.AutoHighLock {
		highAction = RiskActionMedium
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE custom_registration_risk_profiles
		SET risk_score = risk_score + 1,
		    risk_signals = risk_signals || jsonb_build_array($2::text),
		    risk_action = CASE WHEN risk_score + 1 >= $3 THEN $5 WHEN risk_score + 1 >= $4 THEN 'medium' ELSE risk_action END,
		    updated_at = NOW()
		WHERE user_id = $1 AND admin_override = ''`, userID, signal, config.ClusterHigh, config.ClusterMedium, highAction)
	return err
}

func (s *RiskStore) AllowTrialRPM(ctx context.Context, userID int64) error {
	if s == nil || s.redis == nil || userID <= 0 {
		return nil
	}
	config := registrationsettings.Default()
	if s.settings != nil {
		values, err := s.settings.GetMultiple(ctx, registrationsettings.Keys())
		if err != nil {
			return err
		}
		config = registrationsettings.FromValues(values)
	}
	if !config.TrialEnabled {
		return nil
	}
	key := fmt.Sprintf("custom:registration-trial:rpm:%d", userID)
	now := time.Now().UTC().UnixMilli()
	windowStart := now - int64(time.Minute/time.Millisecond)
	result, err := s.redis.Eval(ctx, `
		local key = KEYS[1]
		local now = tonumber(ARGV[1])
		local start = tonumber(ARGV[2])
		local limit = tonumber(ARGV[3])
		redis.call('ZREMRANGEBYSCORE', key, 0, start)
		local count = redis.call('ZCARD', key)
		if count >= limit then
			redis.call('EXPIRE', key, 120)
			return count + 1
		end
		redis.call('ZADD', key, now, ARGV[1] .. ':' .. redis.call('INCR', key .. ':seq'))
		redis.call('EXPIRE', key, 120)
		redis.call('EXPIRE', key .. ':seq', 120)
		return count + 1
	`, []string{key}, now, windowStart, config.TrialRPM).Int64()
	if err != nil {
		return err
	}
	if result > int64(config.TrialRPM) {
		return ErrTrialRPMLimit
	}
	return nil
}

func (s *RiskStore) AllowTrialCost(ctx context.Context, userID int64) error {
	if s == nil || s.db == nil || userID <= 0 {
		return nil
	}
	config := registrationsettings.Default()
	if s.settings != nil {
		values, err := s.settings.GetMultiple(ctx, registrationsettings.Keys())
		if err != nil {
			return err
		}
		config = registrationsettings.FromValues(values)
	}
	if !config.TrialEnabled {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1)`, userID); err != nil {
		return err
	}
	// Usage is written by the gateway after this admission check. Do not add a
	// synthetic reservation here: without a request-lifecycle settlement hook,
	// reservations would accumulate and deny legitimate low-cost users. The
	// advisory lock serializes checks; RPM and concurrency limits cover the
	// short interval before the eventual usage row is committed.
	var used float64
	if err := tx.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(total_cost), 0)
		FROM usage_logs
		WHERE user_id = $1 AND created_at >= NOW() - ($2 * INTERVAL '1 hour')`, userID, config.TrialHours).Scan(&used); err != nil {
		return err
	}
	if used >= config.TrialCostLimit {
		return ErrTrialCostLimit
	}
	return tx.Commit()
}

func (s *RiskStore) TrialActive(ctx context.Context, userID int64) bool {
	if s == nil || s.db == nil || userID <= 0 {
		return false
	}
	config := registrationsettings.Default()
	if s.settings != nil {
		values, err := s.settings.GetMultiple(ctx, registrationsettings.Keys())
		if err != nil {
			return false
		}
		config = registrationsettings.FromValues(values)
	}
	if !config.TrialEnabled {
		return false
	}
	var recent bool
	if err := s.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM custom_registration_cluster_events WHERE user_id = $1 AND registered_at >= NOW() - ($2 * INTERVAL '1 hour'))`, userID, config.TrialHours).Scan(&recent); err != nil || !recent {
		return false
	}
	if !config.PaymentUnlock {
		return true
	}
	var paid float64
	if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(pay_amount - refund_amount), 0) FROM payment_orders WHERE user_id = $1 AND status = 'COMPLETED' AND pay_amount > refund_amount`, userID).Scan(&paid); err != nil {
		return false
	}
	return paid < config.PaymentAmount
}

func hashValue(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func normalizeNetwork(value string) (string, bool) {
	ip := net.ParseIP(NormalizeClientIP(value))
	if ip == nil {
		return "", false
	}
	if v4 := ip.To4(); v4 != nil {
		return fmt.Sprintf("%d.%d.%d.0/24", v4[0], v4[1], v4[2]), true
	}
	// /64 is deliberately conservative and avoids grouping unrelated IPv6
	// allocations more broadly than necessary.
	parts := ip.To16()
	full := net.IP(parts).String()
	chunks := strings.Split(full, ":")
	if len(chunks) < 4 {
		return "", false
	}
	return strings.Join(chunks[:4], ":") + "::/64", true
}
