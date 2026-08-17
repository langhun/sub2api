package custom

import (
	"context"
	"fmt"
	"time"

	activitycheckin "github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/checkin"
	coremiddleware "github.com/Wei-Shaw/sub2api/internal/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

type checkinCaptchaAdapter struct {
	auth *service.AuthService
}

func (a checkinCaptchaAdapter) VerifyCaptchaForAction(
	ctx context.Context,
	proof activitycheckin.CaptchaProof,
	remoteIP string,
	expectedAction string,
) error {
	return a.auth.VerifyCaptchaForAction(ctx, service.CaptchaProof{
		TurnstileToken: proof.TurnstileToken,
		TencentTicket:  proof.TencentTicket,
		TencentRandstr: proof.TencentRandstr,
	}, remoteIP, expectedAction)
}

type checkinLimiterAdapter struct {
	limiter *coremiddleware.RateLimiter
}

func (a checkinLimiterAdapter) Allow(ctx context.Context, key string, limit int, window time.Duration) (activitycheckin.LimitResult, error) {
	result, err := a.limiter.Allow(ctx, key, limit, window)
	return activitycheckin.LimitResult{Allowed: result.Allowed, RetryAfter: result.RetryAfter}, err
}

func newCheckinProtector(authService *service.AuthService, redisClient *redis.Client) (*activitycheckin.Protector, error) {
	if authService == nil {
		return nil, fmt.Errorf("check-in auth service is required")
	}
	if redisClient == nil {
		return nil, fmt.Errorf("check-in redis client is required")
	}
	return activitycheckin.NewProtector(
		checkinCaptchaAdapter{auth: authService},
		checkinLimiterAdapter{limiter: coremiddleware.NewRateLimiter(redisClient)},
	)
}
