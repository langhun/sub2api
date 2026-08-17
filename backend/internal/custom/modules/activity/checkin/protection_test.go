package checkin

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

type captchaVerifierStub struct {
	proof          CaptchaProof
	remoteIP       string
	expectedAction string
	err            error
}

func (s *captchaVerifierStub) VerifyCaptchaForAction(_ context.Context, proof CaptchaProof, remoteIP, expectedAction string) error {
	s.proof = proof
	s.remoteIP = remoteIP
	s.expectedAction = expectedAction
	return s.err
}

type limiterCall struct {
	key    string
	limit  int
	window time.Duration
}

type protectionLimiterStub struct {
	calls   []limiterCall
	results []LimitResult
	err     error
}

func (s *protectionLimiterStub) Allow(_ context.Context, key string, limit int, window time.Duration) (LimitResult, error) {
	s.calls = append(s.calls, limiterCall{key: key, limit: limit, window: window})
	if s.err != nil {
		return LimitResult{}, s.err
	}
	if len(s.results) == 0 {
		return LimitResult{Allowed: true}, nil
	}
	result := s.results[0]
	s.results = s.results[1:]
	return result, nil
}

func TestProtectorVerifiesCaptchaAndAppliesPublicIPLimits(t *testing.T) {
	captcha := &captchaVerifierStub{}
	limiter := &protectionLimiterStub{}
	protector, err := NewProtector(captcha, limiter)
	require.NoError(t, err)

	proof := CaptchaProof{TurnstileToken: "proof-token"}
	err = protector.Verify(context.Background(), 42, "8.8.8.8", proof)

	require.NoError(t, err)
	require.Equal(t, proof, captcha.proof)
	require.Equal(t, "8.8.8.8", captcha.remoteIP)
	require.Equal(t, checkinCaptchaAction, captcha.expectedAction)
	require.Len(t, limiter.calls, 3)
	require.Equal(t, limiterCall{key: "checkin:user:42", limit: userAttemptLimit, window: userAttemptWindow}, limiter.calls[0])
	require.Equal(t, ipAttemptLimit, limiter.calls[1].limit)
	require.Equal(t, ipAttemptWindow, limiter.calls[1].window)
	require.Equal(t, ipDailyLimit, limiter.calls[2].limit)
	require.Equal(t, ipDailyWindow, limiter.calls[2].window)
	require.NotContains(t, limiter.calls[1].key, "8.8.8.8")
}

func TestProtectorRejectsCaptchaAfterShortIPLimit(t *testing.T) {
	captchaErr := errors.New("captcha rejected")
	captcha := &captchaVerifierStub{err: captchaErr}
	limiter := &protectionLimiterStub{}
	protector, err := NewProtector(captcha, limiter)
	require.NoError(t, err)

	err = protector.Verify(context.Background(), 7, "8.8.4.4", CaptchaProof{})

	require.ErrorIs(t, err, captchaErr)
	require.Len(t, limiter.calls, 2)
	require.Equal(t, "checkin:user:7", limiter.calls[0].key)
	require.Contains(t, limiter.calls[1].key, "checkin:ip:short:")
}

func TestNormalizePublicIPGroupsIPv6ByPrefix(t *testing.T) {
	require.Equal(t, "2001:db8:1:2::", normalizePublicIP("2001:db8:1:2::10"))
	require.Equal(t, normalizePublicIP("2001:db8:1:2::10"), normalizePublicIP("2001:db8:1:2:ffff::20"))
	require.NotEqual(t, normalizePublicIP("2001:db8:1:2::10"), normalizePublicIP("2001:db8:1:3::10"))
}

func TestProtectorSkipsSharedProxyIPLimitsForPrivateAddress(t *testing.T) {
	captcha := &captchaVerifierStub{}
	limiter := &protectionLimiterStub{}
	protector, err := NewProtector(captcha, limiter)
	require.NoError(t, err)

	require.NoError(t, protector.Verify(context.Background(), 9, "127.0.0.1", CaptchaProof{}))
	require.Len(t, limiter.calls, 1)
	require.Equal(t, "checkin:user:9", limiter.calls[0].key)
}

func TestProtectorReturnsRateLimitMetadata(t *testing.T) {
	captcha := &captchaVerifierStub{}
	limiter := &protectionLimiterStub{results: []LimitResult{
		{Allowed: true},
		{Allowed: false, RetryAfter: 95 * time.Second},
	}}
	protector, err := NewProtector(captcha, limiter)
	require.NoError(t, err)

	err = protector.Verify(context.Background(), 11, "1.1.1.1", CaptchaProof{})

	require.Equal(t, http.StatusTooManyRequests, infraerrors.Code(err))
	require.Equal(t, "CHECKIN_RATE_LIMITED", infraerrors.Reason(err))
	require.Equal(t, "95", infraerrors.FromError(err).Metadata["retry_after_seconds"])
	retryAfter, ok := retryAfterFromError(err)
	require.True(t, ok)
	require.Equal(t, 95*time.Second, retryAfter)
}

func TestProtectorFailsClosedWhenLimiterIsUnavailable(t *testing.T) {
	limiterErr := errors.New("redis unavailable")
	protector, err := NewProtector(&captchaVerifierStub{}, &protectionLimiterStub{err: limiterErr})
	require.NoError(t, err)

	err = protector.Verify(context.Background(), 12, "1.0.0.1", CaptchaProof{})

	require.Equal(t, http.StatusServiceUnavailable, infraerrors.Code(err))
	require.ErrorIs(t, err, limiterErr)
}
