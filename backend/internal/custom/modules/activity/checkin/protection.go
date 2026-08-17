package checkin

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	checkinCaptchaAction = "checkin"
	userAttemptLimit     = 5
	userAttemptWindow    = 10 * time.Minute
	ipAttemptLimit       = 5
	ipAttemptWindow      = 10 * time.Minute
	ipDailyLimit         = 10
	ipDailyWindow        = 24 * time.Hour
)

var (
	ErrCheckinRateLimited           = infraerrors.TooManyRequests("CHECKIN_RATE_LIMITED", "too many check-in attempts; please try again later")
	ErrCheckinProtectionUnavailable = infraerrors.ServiceUnavailable(
		"CHECKIN_PROTECTION_UNAVAILABLE",
		"check-in protection is temporarily unavailable",
	)
)

type CaptchaProof struct {
	TurnstileToken string
	TencentTicket  string
	TencentRandstr string
}

type CaptchaVerifier interface {
	VerifyCaptchaForAction(ctx context.Context, proof CaptchaProof, remoteIP, expectedAction string) error
}

type LimitResult struct {
	Allowed    bool
	RetryAfter time.Duration
}

type ProtectionLimiter interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration) (LimitResult, error)
}

type CheckinProtector interface {
	Verify(ctx context.Context, userID int64, remoteIP string, proof CaptchaProof) error
}

type Protector struct {
	captcha CaptchaVerifier
	limiter ProtectionLimiter
}

func NewProtector(captcha CaptchaVerifier, limiter ProtectionLimiter) (*Protector, error) {
	if captcha == nil {
		return nil, fmt.Errorf("check-in captcha verifier is required")
	}
	if limiter == nil {
		return nil, fmt.Errorf("check-in rate limiter is required")
	}
	return &Protector{captcha: captcha, limiter: limiter}, nil
}

func (p *Protector) Verify(ctx context.Context, userID int64, remoteIP string, proof CaptchaProof) error {
	if p == nil || p.captcha == nil || p.limiter == nil {
		return ErrCheckinProtectionUnavailable
	}
	if err := p.enforceLimit(ctx, "user:"+strconv.FormatInt(userID, 10), userAttemptLimit, userAttemptWindow); err != nil {
		return err
	}
	publicIP := normalizePublicIP(remoteIP)
	var ipIdentity string
	if publicIP == "" {
		slog.Warn("check-in protection could not resolve a public client IP", "user_id", userID)
	} else {
		ipIdentity = hashLimiterIdentity(publicIP)
		if err := p.enforceLimit(ctx, "ip:short:"+ipIdentity, ipAttemptLimit, ipAttemptWindow); err != nil {
			return err
		}
	}
	if err := p.captcha.VerifyCaptchaForAction(ctx, proof, remoteIP, checkinCaptchaAction); err != nil {
		return err
	}
	if ipIdentity == "" {
		return nil
	}
	return p.enforceLimit(ctx, "ip:daily:"+ipIdentity, ipDailyLimit, ipDailyWindow)
}

func (p *Protector) enforceLimit(ctx context.Context, key string, limit int, window time.Duration) error {
	result, err := p.limiter.Allow(ctx, "checkin:"+key, limit, window)
	if err != nil {
		return ErrCheckinProtectionUnavailable.WithCause(err)
	}
	if result.Allowed {
		return nil
	}
	retryAfter := result.RetryAfter
	if retryAfter <= 0 {
		retryAfter = window
	}
	appErr := ErrCheckinRateLimited.WithMetadata(map[string]string{
		"retry_after_seconds": strconv.FormatInt(roundedSeconds(retryAfter), 10),
	})
	return &rateLimitError{err: appErr, retryAfter: retryAfter}
}

type rateLimitError struct {
	err        error
	retryAfter time.Duration
}

func (e *rateLimitError) Error() string { return e.err.Error() }
func (e *rateLimitError) Unwrap() error { return e.err }

func retryAfterFromError(err error) (time.Duration, bool) {
	var rateErr *rateLimitError
	if !errors.As(err, &rateErr) {
		return 0, false
	}
	return rateErr.retryAfter, true
}

func roundedSeconds(value time.Duration) int64 {
	seconds := int64(value / time.Second)
	if value%time.Second != 0 {
		seconds++
	}
	if seconds < 1 {
		return 1
	}
	return seconds
}

func normalizePublicIP(value string) string {
	parsed := net.ParseIP(strings.TrimSpace(value))
	if parsed == nil || parsed.IsLoopback() || parsed.IsPrivate() || parsed.IsUnspecified() ||
		parsed.IsLinkLocalUnicast() || parsed.IsLinkLocalMulticast() || !parsed.IsGlobalUnicast() {
		return ""
	}
	if ipv4 := parsed.To4(); ipv4 != nil {
		return ipv4.String()
	}
	return parsed.Mask(net.CIDRMask(64, 128)).String()
}

func hashLimiterIdentity(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:12])
}
