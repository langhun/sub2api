package service

import (
	"context"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
)

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

// SmartRoutingStickyCommitter is an optional Overlay hook. Selection and
// sticky persistence are deliberately separate so failed billing/account
// admission cannot pin a session to a group that was never used.
type SmartRoutingStickyCommitter interface {
	CommitSticky(ctx context.Context, apiKeyID, groupID int64, sessionHash string)
}

// SmartRoutingAuditRecorder is implemented by the Overlay audit sink. The
// route decision is created when a group is scored, while the account is only
// known after the scheduler has passed slot and profit admission. Keeping this
// hook optional preserves the upstream service boundary and lets the audit
// writer fill the account field asynchronously.
type SmartRoutingAuditRecorder interface {
	RecordSelectedAccount(ctx context.Context, apiKeyID, groupID, accountID int64)
}

// CommitSmartRoutingAdmission publishes the two side effects that are safe
// only after an account has passed the terminal admission checks. It is kept
// in the service package so every gateway handler (including WebSocket and
// compatibility paths) uses the same ordering.
func CommitSmartRoutingAdmission(ctx context.Context, resolver SmartRoutingResolver, groupID *int64, sessionHash string, accountID int64) {
	if resolver == nil || accountID <= 0 || groupID == nil || *groupID <= 0 {
		return
	}
	key, ok := APIKeyFromContext(ctx)
	if !ok || key == nil || key.ID <= 0 {
		return
	}
	if committer, ok := resolver.(SmartRoutingStickyCommitter); ok {
		committer.CommitSticky(ctx, key.ID, *groupID, sessionHash)
	}
	if recorder, ok := resolver.(SmartRoutingAuditRecorder); ok {
		recorder.RecordSelectedAccount(ctx, key.ID, *groupID, accountID)
	}
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

// WithGroupContext publishes the group selected for the current request.
// Smart routing mutates the request-local API key while it selects a
// candidate, but the authentication middleware's ctxkey.Group value is an
// immutable primary-group snapshot. Downstream mapping, pricing, and endpoint
// checks must see the same selected group as the account scheduler.
func WithGroupContext(ctx context.Context, group *Group) context.Context {
	if ctx == nil || !IsGroupContextValid(group) {
		return ctx
	}
	return context.WithValue(ctx, ctxkey.Group, group)
}

type SmartRoutingRequestState struct {
	// selectedMu protects SmartRoutingSelectedGroups because usage recording is
	// intentionally asynchronous and may inspect the marker while a streaming
	// request is still advancing through another account/group attempt.
	selectedMu     sync.RWMutex
	MainGroupID    int64
	CurrentGroupID int64
	// PrimaryGroupSnapshot is captured before the request-local API key is
	// mutated to a candidate. It lets failover restore a coherent Group
	// pointer even when the primary group repository is temporarily unable to
	// hydrate a fresh snapshot.
	PrimaryGroupSnapshot *Group
	TriedGroups          map[int64]struct{}
	// SmartRoutingSelectedGroups records candidate groups that were actually
	// selected by the Overlay during this request. It is intentionally retained
	// across failover attempts so an asynchronous usage snapshot can still
	// enforce the selected candidate's billing boundary after the mutable
	// request state has moved back to the primary group.
	SmartRoutingSelectedGroups map[int64]struct{}
	// BillingCheckedGroups and BillingRejectedGroups are request-local guards
	// used after the Overlay chooses a candidate.  The auth middleware checks
	// the primary group before routing; candidates need their own subscription,
	// balance and group-RPM check without repeating key/user-wide counters.
	BillingCheckedGroups  map[int64]struct{}
	BillingRejectedGroups map[int64]struct{}
	EndpointType          string
	// ImageGenerationIntent marks a Responses request that explicitly asks for
	// image generation. Candidate group permission must be checked before a
	// group is selected, not only after forwarding starts.
	ImageGenerationIntent bool
	// OpenAI scheduler requirements are populated immediately before the
	// resolver runs. They let the Overlay apply the same account capability
	// gates as the downstream scheduler instead of selecting a group whose pool
	// can never serve this request.
	RequiredOpenAICapability      OpenAIEndpointCapability
	RequiredOpenAIImageCapability OpenAIImagesCapability
	RequiredOpenAITransport       OpenAIUpstreamTransport
	RequireCompact                bool
	// GroupSwitches counts transitions away from the primary/current group.
	// Smart routing uses a small hard limit to prevent an upstream retry loop
	// from walking an unbounded candidate list.
	GroupSwitches    int
	MaxGroupSwitches int
	// SmartRoutingActive is set only after the Overlay has confirmed an
	// enabled smart profile for a supported text HTTP endpoint. It prevents
	// ordinary single-group requests from retrying their selection loop when
	// the account pool is empty.
	SmartRoutingActive bool
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

// MarkSmartRoutingCandidateSelected records a successful candidate admission.
// The marker is request-local and best effort; a nil state is expected on
// legacy service callers that do not participate in smart routing.
func MarkSmartRoutingCandidateSelected(ctx context.Context, groupID int64) {
	if groupID <= 0 {
		return
	}
	state := SmartRoutingStateFromContext(ctx)
	if state == nil {
		return
	}
	state.selectedMu.Lock()
	defer state.selectedMu.Unlock()
	if state.SmartRoutingSelectedGroups == nil {
		state.SmartRoutingSelectedGroups = make(map[int64]struct{})
	}
	state.SmartRoutingSelectedGroups[groupID] = struct{}{}
}

// MarkSmartRoutingGroupExhausted records that the current routed group could
// not admit an account for this request. The scheduler calls this only on an
// ErrNoAvailableAccounts outcome; the next resolver pass can then advance to
// another eligible candidate even when no individual account ID was returned
// (for example, an RPM or capacity gate rejected the whole group).
func MarkSmartRoutingGroupExhausted(ctx context.Context) {
	state := SmartRoutingStateFromContext(ctx)
	if state == nil || !state.SmartRoutingActive || state.CurrentGroupID <= 0 {
		return
	}
	if state.TriedGroups == nil {
		state.TriedGroups = make(map[int64]struct{})
	}
	state.TriedGroups[state.CurrentGroupID] = struct{}{}
}

// ShouldRetrySmartRoutingGroup reports whether a selector error exhausted a
// routed candidate before an account ID could be returned. Handlers use this
// to re-enter the resolver once; primary-group and single-group requests stay
// on their existing immediate-error path.
func ShouldRetrySmartRoutingGroup(ctx context.Context) bool {
	state := SmartRoutingStateFromContext(ctx)
	if state == nil || state.MainGroupID <= 0 || state.CurrentGroupID <= 0 || state.CurrentGroupID == state.MainGroupID {
		return false
	}
	maxSwitches := state.MaxGroupSwitches
	if maxSwitches <= 0 {
		maxSwitches = 2
	}
	if state.GroupSwitches >= maxSwitches {
		return false
	}
	state.selectedMu.RLock()
	_, selected := state.SmartRoutingSelectedGroups[state.CurrentGroupID]
	state.selectedMu.RUnlock()
	if !selected {
		return false
	}
	_, tried := state.TriedGroups[state.CurrentGroupID]
	return tried
}

// IsSmartRoutingCandidateSelected reports whether the API key snapshot points
// to a group that the Overlay actually selected. This avoids treating every
// request state or non-smart fallback group as a candidate billing attempt.
func IsSmartRoutingCandidateSelected(ctx context.Context, apiKey *APIKey) bool {
	if apiKey == nil || apiKey.GroupID == nil {
		return false
	}
	state := SmartRoutingStateFromContext(ctx)
	if state == nil {
		return false
	}
	state.selectedMu.RLock()
	defer state.selectedMu.RUnlock()
	if len(state.SmartRoutingSelectedGroups) == 0 {
		return false
	}
	_, ok := state.SmartRoutingSelectedGroups[*apiKey.GroupID]
	return ok
}

// SmartRoutingPricingAtFromContext returns the request's frozen pricing instant
// shared by the generic and OpenAI token gateways. Routing price protection and
// final usage billing must evaluate peak multipliers at the same instant;
// callers outside a token-priced request receive a current fallback.
func SmartRoutingPricingAtFromContext(ctx context.Context) time.Time {
	if pricingAt := GatewayTokenRequestPricingAtFromContext(ctx); !pricingAt.IsZero() {
		return pricingAt
	}
	if pricingAt := OpenAIPricingAtFromContext(ctx); !pricingAt.IsZero() {
		return pricingAt
	}
	return time.Now()
}
