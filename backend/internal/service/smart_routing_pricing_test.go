package service

import (
	"context"
	"testing"
	"time"
)

func smartRoutingChannelService(groupID int64, platform string) (*ChannelService, *channelCache) {
	cache := newEmptyChannelCache()
	cache.loadedAt = time.Now()
	cache.channelByGroupID[groupID] = &Channel{
		ID:                 7,
		Status:             StatusActive,
		BillingModelSource: BillingModelSourceChannelMapped,
	}
	cache.groupPlatform[groupID] = platform
	svc := &ChannelService{}
	svc.cache.Store(cache)
	return svc, cache
}

func TestCheckSmartRoutingPricingComparable(t *testing.T) {
	const groupID int64 = 42
	ctx := context.Background()

	t.Run("no channel configuration is comparable", func(t *testing.T) {
		svc, _ := smartRoutingChannelService(groupID, PlatformOpenAI)
		ok, err := svc.CheckSmartRoutingPricingComparable(ctx, groupID, "gpt-test")
		if err != nil || !ok {
			t.Fatalf("expected comparable group, ok=%v err=%v", ok, err)
		}
	})

	t.Run("model pricing is fail closed", func(t *testing.T) {
		svc, cache := smartRoutingChannelService(groupID, PlatformOpenAI)
		pricing := &ChannelModelPricing{Platform: PlatformOpenAI, Models: []string{"gpt-test"}, BillingMode: BillingModeToken}
		cache.pricingByGroupModel[channelModelKey{groupID: groupID, platform: PlatformOpenAI, model: "gpt-test"}] = pricing
		ok, err := svc.CheckSmartRoutingPricingComparable(ctx, groupID, "gpt-test")
		if err != nil || ok {
			t.Fatalf("expected model pricing to be rejected, ok=%v err=%v", ok, err)
		}
	})

	t.Run("model mapping is fail closed", func(t *testing.T) {
		svc, cache := smartRoutingChannelService(groupID, PlatformOpenAI)
		cache.mappingByGroupModel[channelModelKey{groupID: groupID, platform: PlatformOpenAI, model: "gpt-test"}] = "gpt-upstream"
		ok, err := svc.CheckSmartRoutingPricingComparable(ctx, groupID, "gpt-test")
		if err != nil || ok {
			t.Fatalf("expected model mapping to be rejected, ok=%v err=%v", ok, err)
		}
	})

	t.Run("empty model rejects configured channel overrides", func(t *testing.T) {
		svc, cache := smartRoutingChannelService(groupID, PlatformOpenAI)
		cache.channelByGroupID[groupID].ModelPricing = []ChannelModelPricing{{Platform: PlatformOpenAI, Models: []string{"*"}}}
		ok, err := svc.CheckSmartRoutingPricingComparable(ctx, groupID, "")
		if err != nil || ok {
			t.Fatalf("expected empty model to be rejected with configured pricing, ok=%v err=%v", ok, err)
		}
	})

	t.Run("restricted channel model is not comparable", func(t *testing.T) {
		svc, cache := smartRoutingChannelService(groupID, PlatformOpenAI)
		cache.channelByGroupID[groupID].RestrictModels = true
		cache.channelByGroupID[groupID].BillingModelSource = BillingModelSourceRequested
		ok, err := svc.CheckSmartRoutingPricingComparable(ctx, groupID, "gpt-restricted")
		if err != nil || ok {
			t.Fatalf("expected restricted model to be rejected, ok=%v err=%v", ok, err)
		}
	})
}

func TestGatewaySmartRoutingPricingComparableDelegates(t *testing.T) {
	const groupID int64 = 42
	svc, cache := smartRoutingChannelService(groupID, PlatformOpenAI)
	cache.mappingByGroupModel[channelModelKey{groupID: groupID, platform: PlatformOpenAI, model: "gpt-test"}] = "gpt-upstream"

	gateway := &GatewayService{channelService: svc}
	ok, err := gateway.CheckSmartRoutingPricingComparable(context.Background(), groupID, "gpt-test")
	if err != nil || ok {
		t.Fatalf("gateway wrapper must delegate fail-closed result, ok=%v err=%v", ok, err)
	}
	openAI := &OpenAIGatewayService{channelService: svc}
	ok, err = openAI.CheckSmartRoutingPricingComparable(context.Background(), groupID, "gpt-test")
	if err != nil || ok {
		t.Fatalf("openai wrapper must delegate fail-closed result, ok=%v err=%v", ok, err)
	}
}
