package service

import "context"

// SmartRouteDecision is the request-local group decision produced by the
// Overlay. The gateway keeps all account-level scheduling in the existing
// scheduler and only asks the Overlay which compatible group to use.
type SmartRouteDecision struct {
	GroupID       int64
	FallbackDepth int
	Reason        string
}

// SmartRoutingResolver is implemented by the custom Overlay. Keeping this
// interface in the service package avoids importing custom code into upstream
// gateway packages.
type SmartRoutingResolver interface {
	Resolve(ctx context.Context, apiKey *APIKey, requestedModel, sessionHash string, excludedAccountIDs map[int64]struct{}) (*SmartRouteDecision, error)
}

type apiKeyContextKey struct{}
type smartRoutingStateContextKey struct{}

// WithAPIKeyContext makes the authenticated key available to shared gateway
// services without coupling them to gin.Context.
func WithAPIKeyContext(ctx context.Context, key *APIKey) context.Context {
	return context.WithValue(ctx, apiKeyContextKey{}, key)
}

// APIKeyFromContext returns the request-local authenticated key.
func APIKeyFromContext(ctx context.Context) (*APIKey, bool) {
	key, ok := ctx.Value(apiKeyContextKey{}).(*APIKey)
	return key, ok && key != nil
}

type SmartRoutingRequestState struct {
	MainGroupID    int64
	CurrentGroupID int64
	TriedGroups    map[int64]struct{}
	EndpointType   string
	// GroupSwitches counts transitions away from the primary/current group.
	// Smart routing uses a small hard limit to prevent an upstream retry loop
	// from walking an unbounded candidate list.
	GroupSwitches    int
	MaxGroupSwitches int
	// ContinuityLocked is set for requests carrying previous_response_id.
	// Such requests must stay in the primary group; silently moving the
	// response chain to another group would produce an upstream 404/401 or,
	// worse, a semantically different conversation.
	ContinuityLocked bool
}

// WithSmartRoutingState stores mutable request-local state. A pointer is used
// deliberately so repeated selector calls can advance to the next group.
func WithSmartRoutingState(ctx context.Context, state *SmartRoutingRequestState) context.Context {
	return context.WithValue(ctx, smartRoutingStateContextKey{}, state)
}

func SmartRoutingStateFromContext(ctx context.Context) *SmartRoutingRequestState {
	state, _ := ctx.Value(smartRoutingStateContextKey{}).(*SmartRoutingRequestState)
	return state
}
