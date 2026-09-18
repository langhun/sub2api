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
	selectedMu          sync.RWMutex
	MainGroupID         int64
	CurrentGroupID      int64
	ExhaustedGroups     map[int64]struct{}
	FailedAccountIDs    map[int64]struct{}
	NoRoute             bool
	Strategy            string
	SpeedWeight         float64
	SuccessWeight       float64
	RequestedModel      string
	Stream              bool
	Attempt             int
	CapacityRetries     int
	Observer            SmartRoutingAttemptObserver
	ContinuityGroupID   int64
	ContinuityAccountID int64
	RecoveryProbe       bool
	AccountMetrics      map[int64]SmartRoutingAccountMetric
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
	// SmartRoutingPrimaryRetryUsed prevents an exhausted primary group from
	// re-entering the resolver forever when every candidate is unavailable and
	// the resolver restores the primary without increasing GroupSwitches.
	SmartRoutingPrimaryRetryUsed bool
	// SmartRoutingActive is set only after the Overlay has confirmed an
	// enabled smart profile for a supported text HTTP endpoint. It prevents
	// ordinary single-group requests from retrying their selection loop when
	// the account pool is empty.
	SmartRoutingActive bool
	// ContractInvalid is set when the primary group contract changes (platform,
	// subscription type, status, or identity) while a smart-routed request is
	// in flight. Restoring the frozen snapshot keeps billing metadata coherent,
	// but must not allow downstream schedulers to reload the changed row by ID.
	ContractInvalid bool
	// ContinuityLocked is set for requests carrying previous_response_id.
	// Such requests must stay with their verified response owner; moving the
	// response chain to another group would produce an upstream 404/401 or,
	// worse, a semantically different conversation.
	ContinuityLocked bool
}

// SmartRoutingContractValid is the request-level fail-closed gate. Ordinary
// single-group requests have no active smart-routing state and remain
// unchanged; once smart routing is active, a detected contract race aborts
// before any platform/account pool is resolved.
func SmartRoutingContractValid(ctx context.Context) bool {
	state := SmartRoutingStateFromContext(ctx)
	return state == nil || (!state.ContractInvalid && !state.NoRoute)
}

func MarkSmartRoutingContractInvalid(ctx context.Context) {
	if state := SmartRoutingStateFromContext(ctx); state != nil && state.SmartRoutingActive {
		state.ContractInvalid = true
	}
}

// SmartRoutingGroupsCompatible is the shared hard boundary for a routed
// candidate. A candidate may only use the same concrete upstream platform and
// billing/subscription type as the authenticated primary group. Composite
// groups are deliberately excluded because their target platform is resolved
// per request and therefore cannot provide the same protocol or price
// contract. Keep this check independent of scoring, permissions, exploration,
// or billing so no later decision can widen the candidate pool.
func SmartRoutingGroupsCompatible(primary, candidate *Group) bool {
	if primary == nil || candidate == nil {
		return false
	}
	// A routed candidate is admitted only while it is active. The ranker also
	// applies this filter, but the resolver performs a second check after
	// hydrating the group to close the rank-to-dispatch TOCTOU window. The
	// primary group is intentionally not checked here: it is the authenticated
	// request's existing fallback target and normal group/account scheduling
	// remains responsible for its availability.
	if candidate.Status != StatusActive {
		return false
	}
	if primary.Platform == "" || candidate.Platform == "" {
		return false
	}
	if primary.Platform == PlatformComposite || candidate.Platform == PlatformComposite {
		return false
	}
	if primary.Platform != candidate.Platform {
		return false
	}
	return primary.SubscriptionType == candidate.SubscriptionType
}

// SmartRoutingPrimaryContractMatches verifies the primary group immediately
// before a routed candidate is admitted.  Authentication captures a request
// snapshot, while administrators can edit a group concurrently; if its
// platform or subscription type changed, the old routing decision is stale and
// must fall back to the primary path.  A nil snapshot is tolerated for legacy
// internal callers, but the freshly loaded primary still has to be concrete,
// active, and non-composite.
func SmartRoutingPrimaryContractMatches(snapshot, current *Group, primaryID int64) bool {
	if current == nil || primaryID <= 0 || current.ID != primaryID || current.Status != StatusActive {
		return false
	}
	if current.Platform == "" || current.Platform == PlatformComposite {
		return false
	}
	if snapshot == nil {
		return true
	}
	if snapshot.ID != primaryID || snapshot.Platform == "" || snapshot.Platform == PlatformComposite {
		return false
	}
	return snapshot.Platform == current.Platform && snapshot.SubscriptionType == current.SubscriptionType
}

// SmartRoutingPrimaryRestoreGroup chooses the request-local group object to
// publish when a candidate route is rejected.  A fresh database row is only
// safe when it still matches the ingress snapshot's protocol and billing
// contract.  If an administrator changed the primary while the request was
// in flight, retain the frozen snapshot instead of leaking the new platform
// (for example Gemini) into an OpenAI request.  Legacy callers without a
// snapshot may still use a concrete fresh primary as their best available
// restore value.
func SmartRoutingPrimaryRestoreGroup(snapshot, current *Group, primaryID int64) *Group {
	if primaryID <= 0 {
		return nil
	}
	if snapshot != nil && snapshot.ID == primaryID {
		if current != nil && SmartRoutingPrimaryContractMatches(snapshot, current, primaryID) {
			return current
		}
		return snapshot
	}
	if current != nil && current.ID == primaryID {
		return current
	}
	return nil
}

// SmartRoutingResolvedGroupAllowed is the final admission check after legacy
// group resolution (for example Claude-Code fallback traversal) has hydrated
// a group from the repository.  Comparing IDs alone is insufficient: the
// primary row may have been edited in place from OpenAI to Gemini while a
// request was in flight.  Keep ordinary single-group callers unchanged by
// returning true when smart routing is inactive.
func SmartRoutingResolvedGroupAllowed(ctx context.Context, group *Group, groupID *int64) bool {
	state := SmartRoutingStateFromContext(ctx)
	if state == nil || !state.SmartRoutingActive {
		return true
	}
	if group == nil || groupID == nil || group.ID != *groupID || state.PrimaryGroupSnapshot == nil {
		return false
	}
	if state.MainGroupID > 0 && *groupID == state.MainGroupID {
		return SmartRoutingPrimaryContractMatches(state.PrimaryGroupSnapshot, group, state.MainGroupID)
	}
	return SmartRoutingGroupsCompatible(state.PrimaryGroupSnapshot, group)
}

// WithSmartRoutingState stores mutable request-local state. A pointer is used
// deliberately so repeated selector calls can advance to the next group.
func WithSmartRoutingState(ctx context.Context, state *SmartRoutingRequestState) context.Context {
	return context.WithValue(ctx, smartRoutingStateContextKey{}, state)
}

func SmartRoutingStateFromContext(ctx context.Context) *SmartRoutingRequestState {
	if ctx == nil {
		return nil
	}
	state, _ := ctx.Value(smartRoutingStateContextKey{}).(*SmartRoutingRequestState)
	return state
}

// SmartRoutingGroupResolutionChanged reports whether a downstream legacy
// resolver changed the group selected by the smart-routing overlay.  A
// Claude-Code fallback (or any future implicit fallback) is allowed for
// ordinary single-group requests, but it must never widen a smart route to a
// group that was not selected and hard-filtered by the overlay.  Callers use
// this as a final guard after the legacy group resolver has run.
func SmartRoutingGroupResolutionChanged(ctx context.Context, selectedGroupID, resolvedGroupID *int64) bool {
	state := SmartRoutingStateFromContext(ctx)
	if state == nil || !state.SmartRoutingActive {
		return false
	}
	if selectedGroupID == nil || resolvedGroupID == nil {
		return true
	}
	return *selectedGroupID != *resolvedGroupID
}

// RestoreSmartRoutingPrimarySnapshot puts the request-local API key back on
// the authenticated primary after a downstream legacy resolver (for example
// Claude-Code fallback) attempts to change the selected group. This is a
// defensive billing/mapping invariant: returning an error while leaving
// apiKey.Group pointed at the candidate can make deferred usage recording use
// the wrong platform or subscription.
func RestoreSmartRoutingPrimarySnapshot(ctx context.Context, key *APIKey) {
	if key == nil {
		return
	}
	state := SmartRoutingStateFromContext(ctx)
	if state == nil || state.MainGroupID <= 0 {
		return
	}
	primaryID := state.MainGroupID
	if state.PrimaryGroupSnapshot != nil && state.PrimaryGroupSnapshot.ID == primaryID {
		key.Group = state.PrimaryGroupSnapshot
	} else if key.Group == nil || key.Group.ID != primaryID {
		// Do not retain a candidate pointer next to a primary ID when no trusted
		// snapshot is available. The caller will surface the routing error and
		// downstream billing must fail closed rather than charge the candidate.
		key.Group = nil
	}
	key.GroupID = &primaryID
	state.CurrentGroupID = primaryID
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
	if state == nil || !state.SmartRoutingActive || state.CurrentGroupID <= 0 || state.ContinuityLocked {
		// Continuity-locked requests (for example Responses requests carrying
		// previous_response_id) are never eligible for cross-group replay. Do
		// not mark the group exhausted because that marker can leak into a later
		// retry decision on the same request context.
		return
	}
	if state.TriedGroups == nil {
		state.TriedGroups = make(map[int64]struct{})
	}
	state.TriedGroups[state.CurrentGroupID] = struct{}{}
	if state.ExhaustedGroups == nil {
		state.ExhaustedGroups = make(map[int64]struct{})
	}
	state.ExhaustedGroups[state.CurrentGroupID] = struct{}{}
}

// TrySmartRoutingGroupFailover advances the request to another configured
// group after an upstream account failover budget or account-selection
// exhaustion. A nil failoverErr is valid when the scheduler exhausted its
// exclusion set without an upstream error (for example, every account was
// rejected by a profit gate); the active smart-routing state is still the
// authority for whether a group switch is allowed. It is kept as a single
// guard so every HTTP gateway applies the same billing and streaming boundary:
// a non-retryable error, a cancelled request, or any bytes written to the
// client must remain on the current group and terminate normally.
//
// The caller must reset its account-level failover state when this returns
// true. Account IDs and retry counters are scoped to the group that just
// failed; carrying them into a candidate group can incorrectly exclude a
// healthy account or skip its first retry budget.
func TrySmartRoutingGroupFailover(ctx context.Context, failoverErr *UpstreamFailoverError, outputStarted bool) bool {
	if outputStarted || ctx == nil || ctx.Err() != nil {
		return false
	}
	if failoverErr != nil && !failoverErr.ShouldRetryNextAccount() {
		return false
	}
	MarkSmartRoutingGroupExhausted(ctx)
	return ShouldRetrySmartRoutingGroup(ctx)
}

// ShouldRetrySmartRoutingGroup reports whether a selector error exhausted the
// current group before an account ID could be returned. It is enabled only
// after the resolver has confirmed an active smart profile, so ordinary
// single-group requests keep their existing immediate-error path. The primary
// group may enter the resolver once while switch budget remains; after the
// resolver has exhausted the candidate budget, the final primary restore does
// not re-enter the loop.
func ShouldRetrySmartRoutingGroup(ctx context.Context) bool {
	state := SmartRoutingStateFromContext(ctx)
	if state == nil || !state.SmartRoutingActive || state.NoRoute || state.ContractInvalid || state.MainGroupID <= 0 || state.CurrentGroupID <= 0 {
		return false
	}
	maxSwitches := state.MaxGroupSwitches
	if maxSwitches <= 0 {
		maxSwitches = 2
	}
	// At the switch budget's boundary the resolver still needs one pass to
	// return to the primary group as the documented safe degradation target.
	// A primary selection is not another candidate switch.
	if state.GroupSwitches > maxSwitches {
		return false
	}
	_, tried := state.TriedGroups[state.CurrentGroupID]
	if !tried || state.ContinuityLocked {
		return false
	}
	if state.CurrentGroupID == state.MainGroupID {
		if state.SmartRoutingPrimaryRetryUsed || state.GroupSwitches >= maxSwitches {
			return false
		}
		state.SmartRoutingPrimaryRetryUsed = true
		return true
	}
	state.selectedMu.RLock()
	_, selected := state.SmartRoutingSelectedGroups[state.CurrentGroupID]
	state.selectedMu.RUnlock()
	return selected
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
