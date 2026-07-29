package service

import "context"

// OpenAIAccountSchedulingPolicy is an optional selection hint supplied at the
// application composition root. It deliberately exposes no Overlay-specific
// concepts: the gateway only asks whether an otherwise eligible account should
// be preferred for a new OpenAI-compatible request.
//
// Implementations must be non-blocking because this method runs on the gateway
// scheduling path.
type OpenAIAccountSchedulingPolicy interface {
	PreferOpenAIAccount(ctx context.Context, groupID *int64, requestedModel string, accountID int64) bool
}

// SetOpenAIAccountSchedulingPolicy installs an optional policy after the base
// service has been built. This is an explicit composition hook so upstream
// gateway construction remains independent of application-specific modules.
func (s *OpenAIGatewayService) SetOpenAIAccountSchedulingPolicy(policy OpenAIAccountSchedulingPolicy) {
	if s == nil {
		return
	}
	s.openAIAccountSchedulingPolicy = policy
}

func (s *OpenAIGatewayService) prefersOpenAIAccount(ctx context.Context, groupID *int64, requestedModel string, accountID int64) bool {
	return s != nil && s.openAIAccountSchedulingPolicy != nil &&
		s.openAIAccountSchedulingPolicy.PreferOpenAIAccount(ctx, groupID, requestedModel, accountID)
}
