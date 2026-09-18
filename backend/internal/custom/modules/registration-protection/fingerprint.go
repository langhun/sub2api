package registrationprotection

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	// ReservationTTL bounds the lifetime of a registration reservation. A
	// crashed or abandoned registration therefore cannot block a fingerprint
	// forever, while normal email/OAuth flows have ample time to finish.
	ReservationTTL = 15 * time.Minute

	defaultRegistrationSource = "unknown"
)

var (
	ErrUnavailable         = errors.New("registration fingerprint store is unavailable")
	ErrInvalidReservation  = errors.New("invalid registration fingerprint reservation")
	ErrReservationFinished = errors.New("registration fingerprint reservation is already finished")
)

// BeginInput describes the request metadata used for reward risk evaluation.
// UserAgent is normalized and hashed immediately; the raw value is never
// written to the database or included in errors.
type BeginInput struct {
	ClientIP  string
	UserAgent string
	DeviceID  string
	Source    string
}

// CommitResult reports whether this registration should receive signup
// rewards. RewardBlocked is true for every registration after the first
// successful registration for a fingerprint, including concurrent races.
type CommitResult struct {
	FingerprintHash   string
	RewardBlocked     bool
	FirstRegistration bool
}

// Reservation is an opaque registration reservation returned by Begin.
// Callers must Commit after the user is created or Abort if creation fails.
type Reservation struct {
	store            *Store
	fingerprintHash  string
	reservationToken string
	source           string
	rewardBlocked    bool
	noFingerprint    bool

	mu        sync.Mutex
	finished  bool
	committed *CommitResult
}

// RewardBlocked reports the decision known at Begin time. Commit returns the
// final decision, which may change to blocked if another request wins a race.
func (r *Reservation) RewardBlocked() bool {
	if r == nil {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.rewardBlocked
}

// FingerprintHash returns the opaque SHA-256 identifier, useful for internal
// diagnostics without exposing client metadata.
func (r *Reservation) FingerprintHash() string {
	if r == nil {
		return ""
	}
	return r.fingerprintHash
}

// Store persists registration fingerprint history and short-lived
// reservations. database/sql is safe for concurrent use; transaction-level
// locks and unique constraints provide cross-process serialization.
type Store struct {
	db *sql.DB
}

// Repository is retained as a descriptive alias for callers that use the
// repository naming convention in other Overlay modules.
type Repository = Store

func NewStore(db *sql.DB) *Store { return &Store{db: db} }

func NewFingerprintStore(db *sql.DB) *Store { return NewStore(db) }

func NewRepository(db *sql.DB) *Store { return NewStore(db) }

// Begin reserves a previously unseen fingerprint. Existing history and a
// concurrent reservation both produce a blocked reservation, while still
// allowing the account itself to be created.
func (s *Store) Begin(ctx context.Context, input BeginInput) (*Reservation, error) {
	if s == nil || s.db == nil {
		return nil, ErrUnavailable
	}
	hash, ok := FingerprintHashWithDevice(input.ClientIP, input.UserAgent, input.DeviceID)
	reservation := &Reservation{
		store:           s,
		fingerprintHash: hash,
		source:          normalizeSource(input.Source),
		noFingerprint:   !ok,
	}
	if !ok {
		return reservation, nil
	}

	// Serializable isolation closes the check-then-reserve race with a
	// concurrent commit. If PostgreSQL aborts a serialization conflict, the
	// AuthService treats the risk decision as fail-closed for rewards while
	// allowing the account itself to continue.
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, fmt.Errorf("begin registration fingerprint reservation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var exists bool
	if err := tx.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM custom_registration_fingerprint_history
			WHERE fingerprint_hash = $1
		)`, hash).Scan(&exists); err != nil {
		return nil, fmt.Errorf("check registration fingerprint history: %w", err)
	}
	if exists {
		reservation.rewardBlocked = true
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("commit blocked fingerprint reservation: %w", err)
		}
		return reservation, nil
	}

	// Remove abandoned reservations before attempting the unique insert. A
	// concurrent transaction still wins the unique key and makes this request
	// blocked, which is the conservative reward decision.
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM custom_registration_fingerprint_reservations
		WHERE fingerprint_hash = $1 AND expires_at <= NOW()`, hash); err != nil {
		return nil, fmt.Errorf("expire registration fingerprint reservation: %w", err)
	}
	token := uuid.NewString()
	var storedToken string
	err = tx.QueryRowContext(ctx, `
		INSERT INTO custom_registration_fingerprint_reservations
		    (fingerprint_hash, reservation_token, source, expires_at)
		VALUES ($1, $2, $3, NOW() + INTERVAL '15 minutes')
		ON CONFLICT (fingerprint_hash) DO NOTHING
		RETURNING reservation_token`, hash, token, reservation.source).Scan(&storedToken)
	if errors.Is(err, sql.ErrNoRows) {
		reservation.rewardBlocked = true
	} else if err != nil {
		return nil, fmt.Errorf("reserve registration fingerprint: %w", err)
	} else {
		reservation.reservationToken = storedToken
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit registration fingerprint reservation: %w", err)
	}
	return reservation, nil
}

// Commit records a successful account creation and returns its final reward
// eligibility. The event row is retained even if the account is later soft
// deleted, preserving the historical signal for future registrations.
func (s *Store) Commit(ctx context.Context, reservation *Reservation, userID int64) (CommitResult, error) {
	if s == nil || s.db == nil {
		return CommitResult{}, ErrUnavailable
	}
	if reservation == nil || reservation.store != s || (reservation.fingerprintHash == "" && !reservation.noFingerprint) || userID <= 0 {
		return CommitResult{}, ErrInvalidReservation
	}
	reservation.mu.Lock()
	if reservation.finished {
		if reservation.committed != nil {
			result := *reservation.committed
			reservation.mu.Unlock()
			return result, nil
		}
		reservation.mu.Unlock()
		return CommitResult{}, ErrReservationFinished
	}
	reservation.finished = true
	reservation.mu.Unlock()
	if reservation.noFingerprint {
		result := CommitResult{FirstRegistration: true}
		reservation.mu.Lock()
		reservation.rewardBlocked = result.RewardBlocked
		reservation.committed = &result
		reservation.mu.Unlock()
		return result, nil
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		reservation.resetFinished()
		return CommitResult{}, fmt.Errorf("begin registration fingerprint commit: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if reservation.reservationToken != "" {
		result, err := tx.ExecContext(ctx, `
			DELETE FROM custom_registration_fingerprint_reservations
			WHERE fingerprint_hash = $1 AND reservation_token = $2`, reservation.fingerprintHash, reservation.reservationToken)
		if err != nil {
			reservation.resetFinished()
			return CommitResult{}, fmt.Errorf("consume registration fingerprint reservation: %w", err)
		}
		_, err = result.RowsAffected()
		if err != nil {
			reservation.resetFinished()
			return CommitResult{}, fmt.Errorf("read consumed registration fingerprint reservation: %w", err)
		}
		// Zero deleted rows are allowed for expired/aborted reservations.
		// The history upsert below determines whether this is a repeat.
	}

	var count int64
	if err := tx.QueryRowContext(ctx, `
		INSERT INTO custom_registration_fingerprint_history
		    (fingerprint_hash, first_user_id, first_source, registration_count, last_registered_at)
		VALUES ($1, $2, $3, 1, NOW())
		ON CONFLICT (fingerprint_hash) DO UPDATE
		SET registration_count = custom_registration_fingerprint_history.registration_count + 1,
		    last_registered_at = NOW()
		RETURNING registration_count`, reservation.fingerprintHash, userID, reservation.source).Scan(&count); err != nil {
		reservation.resetFinished()
		return CommitResult{}, fmt.Errorf("record registration fingerprint history: %w", err)
	}
	repeat := count > 1 || reservation.rewardBlocked
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO custom_registration_fingerprint_events
		    (fingerprint_hash, user_id, source, is_repeat, reward_blocked)
		VALUES ($1, $2, $3, $4, $5)`, reservation.fingerprintHash, userID, reservation.source, repeat, repeat); err != nil {
		reservation.resetFinished()
		return CommitResult{}, fmt.Errorf("record registration fingerprint event: %w", err)
	}
	if err := tx.Commit(); err != nil {
		reservation.resetFinished()
		return CommitResult{}, fmt.Errorf("commit registration fingerprint history: %w", err)
	}
	result := CommitResult{FingerprintHash: reservation.fingerprintHash, RewardBlocked: repeat, FirstRegistration: !repeat}
	reservation.mu.Lock()
	reservation.rewardBlocked = result.RewardBlocked
	reservation.committed = &result
	reservation.mu.Unlock()
	return result, nil
}

// Abort releases an uncommitted reservation. No history is written, so a
// failed registration cannot poison the fingerprint permanently.
func (s *Store) Abort(ctx context.Context, reservation *Reservation) error {
	if s == nil || s.db == nil {
		return ErrUnavailable
	}
	if reservation == nil || reservation.store != s || (reservation.fingerprintHash == "" && !reservation.noFingerprint) {
		return ErrInvalidReservation
	}
	reservation.mu.Lock()
	if reservation.finished {
		reservation.mu.Unlock()
		return nil
	}
	reservation.finished = true
	reservation.mu.Unlock()
	if reservation.noFingerprint || reservation.reservationToken == "" {
		return nil
	}
	if _, err := s.db.ExecContext(ctx, `
		DELETE FROM custom_registration_fingerprint_reservations
		WHERE fingerprint_hash = $1 AND reservation_token = $2`, reservation.fingerprintHash, reservation.reservationToken); err != nil {
		reservation.resetFinished()
		return fmt.Errorf("abort registration fingerprint reservation: %w", err)
	}
	return nil
}

// IsRewardBlocked answers whether a committed user belongs to a repeated
// fingerprint. Soft-deleted users keep their event row (the user foreign key
// becomes NULL only on hard deletion), so the historical decision remains
// available for active accounts and audit tooling.
func (s *Store) IsRewardBlocked(ctx context.Context, userID int64) (bool, error) {
	if s == nil || s.db == nil {
		return false, ErrUnavailable
	}
	if userID <= 0 {
		return false, ErrInvalidReservation
	}
	var blocked bool
	err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(BOOL_OR(reward_blocked), FALSE)
		FROM custom_registration_fingerprint_events
		WHERE user_id = $1`, userID).Scan(&blocked)
	if err != nil {
		return false, fmt.Errorf("check registration reward eligibility: %w", err)
	}
	return blocked, nil
}

func (r *Reservation) resetFinished() {
	r.mu.Lock()
	r.finished = false
	r.mu.Unlock()
}

// NormalizeUserAgent collapses insignificant whitespace and normalizes case.
// It intentionally keeps product/version tokens intact so different clients
// are not accidentally treated as the same device.
func NormalizeUserAgent(value string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(value))), " ")
}

// NormalizeClientIP canonicalizes valid IP literals and otherwise uses a
// lower-cased, trimmed representation. Hostnames are not resolved.
func NormalizeClientIP(value string) string {
	value = strings.TrimSpace(value)
	if parsed := net.ParseIP(value); parsed != nil {
		return parsed.String()
	}
	return strings.ToLower(value)
}

// FingerprintHash computes the opaque identifier used by the persistence
// layer. Empty metadata is not recorded because it cannot distinguish clients.
func FingerprintHash(clientIP, userAgent string) (string, bool) {
	return FingerprintHashWithDevice(clientIP, userAgent, "")
}

// FingerprintHashWithDevice adds an optional opaque device token to the
// fingerprint. IP and User-Agent remain separately scored by cluster risk;
// this value only improves correlation when a client rotates either signal.
func FingerprintHashWithDevice(clientIP, userAgent, deviceID string) (string, bool) {
	ip := NormalizeClientIP(clientIP)
	ua := NormalizeUserAgent(userAgent)
	if ip == "" || ua == "" {
		return "", false
	}
	device := strings.TrimSpace(deviceID)
	sum := sha256.Sum256([]byte(ip + "\x00" + ua + "\x00" + device))
	return hex.EncodeToString(sum[:]), true
}

func normalizeSource(source string) string {
	source = strings.TrimSpace(strings.ToLower(source))
	if source == "" {
		return defaultRegistrationSource
	}
	if len(source) > 80 {
		return source[:80]
	}
	return source
}
