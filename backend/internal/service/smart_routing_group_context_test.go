package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/stretchr/testify/require"
)

func TestWithGroupContextPublishesOnlyHydratedGroups(t *testing.T) {
	primary := &Group{ID: 1, Platform: PlatformOpenAI, Status: StatusActive, Hydrated: true}
	candidate := &Group{ID: 2, Platform: PlatformOpenAI, Status: StatusActive, Hydrated: true}
	ctx := context.WithValue(context.Background(), ctxkey.Group, primary)

	selected := WithGroupContext(ctx, candidate)
	got, ok := selected.Value(ctxkey.Group).(*Group)
	require.True(t, ok)
	require.Same(t, candidate, got)

	// An untrusted/incomplete group must not replace the authenticated group
	// context, otherwise downstream pricing and endpoint checks would lose the
	// primary fallback when the candidate lookup is incomplete.
	invalid := &Group{ID: candidate.ID, Platform: candidate.Platform, Status: candidate.Status}
	unchanged := WithGroupContext(ctx, invalid)
	got, ok = unchanged.Value(ctxkey.Group).(*Group)
	require.True(t, ok)
	require.Same(t, primary, got)
}

func TestOpenAIGroupPricingContextPreservesFrozenPricingAt(t *testing.T) {
	svc := &OpenAIGatewayService{}
	groupID := int64(2)
	pricingAt := time.Date(2026, time.January, 15, 8, 30, 0, 0, time.UTC)

	ctx := svc.WithOpenAIGroupPricingContext(context.Background(), &groupID, pricingAt)
	require.Equal(t, pricingAt, OpenAIPricingAtFromContext(ctx))
}

func TestGatewayGroupPricingContextRefreshesBillingGroup(t *testing.T) {
	svc := &GatewayService{}
	groupID := int64(2)
	candidate := &Group{ID: groupID, Platform: PlatformAnthropic, Status: StatusActive, Hydrated: true}
	pricingAt := time.Date(2026, time.January, 15, 8, 30, 0, 0, time.UTC)

	ctx := WithGroupContext(context.Background(), candidate)
	ctx = svc.WithGatewayGroupPricingContext(ctx, &groupID, pricingAt)
	require.Equal(t, pricingAt, GatewayTokenRequestPricingAtFromContext(ctx))
	require.Same(t, candidate, gatewayTokenRequestBillingGroupFromContext(ctx))
}

func TestSmartRoutingPricingAtUsesFrozenGatewayTimestamp(t *testing.T) {
	pricingAt := time.Date(2026, time.February, 2, 12, 0, 0, 0, time.UTC)
	ctx, _ := WithGatewayTokenRequestPricing(context.Background())
	ctx = context.WithValue(ctx, gatewayTokenRequestPricingAtCtxKey{}, pricingAt)
	require.Equal(t, pricingAt, SmartRoutingPricingAtFromContext(ctx))
}
