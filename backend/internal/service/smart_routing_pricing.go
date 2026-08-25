package service

import "context"

// CheckSmartRoutingPricingComparable is the gateway-facing read-only pricing
// guard. A nil channel service means the group has no channel-level pricing
// override in this process and therefore remains comparable by multiplier.
func (s *GatewayService) CheckSmartRoutingPricingComparable(ctx context.Context, groupID int64, model string) (bool, error) {
	if s == nil || s.channelService == nil {
		return true, nil
	}
	return s.channelService.CheckSmartRoutingPricingComparable(ctx, groupID, model)
}

// CheckSmartRoutingPricingComparable is the OpenAI gateway equivalent of the
// generic gateway guard. Keep this wrapper separate so the Overlay does not
// need to know which upstream scheduler owns the request.
func (s *OpenAIGatewayService) CheckSmartRoutingPricingComparable(ctx context.Context, groupID int64, model string) (bool, error) {
	if s == nil || s.channelService == nil {
		return true, nil
	}
	return s.channelService.CheckSmartRoutingPricingComparable(ctx, groupID, model)
}
