package service

import (
	"context"
	"net/http"
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

type SmartRoutingOutcome struct {
	RequestID      string
	UsageRequestID string
	APIKeyID       int64
	FinalGroupID   int64
	FinalAccountID int64
	FinalSuccess   bool
	Attempts       int
	GroupSwitches  int
	DurationMs     int64
	FirstTokenMs   int64
	ErrorStatus    int
}

type SmartRoutingOutcomeRecorder interface {
	RecordRoutingOutcome(SmartRoutingOutcome)
}

// SmartRoutingSameAccountRetryPolicy lets the mounted Overlay veto a pool
// retry without placing provider-specific error interpretation in core.
type SmartRoutingSameAccountRetryPolicy interface {
	SuppressSameAccountRetry(*UpstreamFailoverError) bool
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

type smartRoutingSiteAttemptKey struct {
	groupID   int64
	accountID int64
}

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
	selectedMu       sync.RWMutex
	attemptMu        sync.Mutex
	turnAttemptCount int
	turnAttemptStart time.Time
	siteAttempts     map[smartRoutingSiteAttemptKey]int
	MainGroupID      int64
	CurrentGroupID   int64
	ExhaustedGroups  map[int64]struct{}
	FailedAccountIDs map[int64]struct{}
	// FailedAccountGroups scopes request-local account failures to the group
	// where they occurred. The same upstream website/account ID may legally
	// be mounted in multiple candidate groups, so a flat failed-account set
	// would incorrectly quarantine it after a cross-group switch.
	FailedAccountGroups map[int64]map[int64]struct{}
	AccountExclusions   map[int64]string
	NoRoute             bool
	Strategy            string
	PriceWeight         float64
	SpeedWeight         float64
	SuccessWeight       float64
	RequestedModel      string
	Stream              bool
	Attempt             int
	// TotalAttempts counts every upstream forward started by this request,
	// including attempts whose error is not eligible for telemetry. It is the
	// hard request-level safety budget; Attempt remains the historical
	// telemetry counter.
	TotalAttempts         int
	SmartRoutingStartedAt time.Time
	// CurrentAttemptStartedAt is refreshed immediately before each actual
	// upstream forward. It is used with the metric windows below to convert a
	// per-attempt TTFT into a cumulative group or website sample.
	CurrentAttemptStartedAt time.Time
	// RequestElapsedMs is the latest request-level elapsed sample observed by
	// the route. It is kept for outcome diagnostics and is never written as a
	// website/account speed sample after a cross-account or cross-group switch.
	RequestElapsedMs int64
	// GroupMetricStartedAt starts a cumulative speed window for the current
	// group. It survives account changes inside one group so a group sample
	// includes same-group failover, pool retries and their backoff.
	GroupMetricStartedAt time.Time
	GroupMetricGroupID   int64
	// AccountMetricStartedAt starts a cumulative speed window for the current
	// website/account entry. It survives pool retries for the same account but
	// resets when the scheduler moves to another website or group.
	AccountMetricStartedAt  time.Time
	AccountMetricGroupID    int64
	AccountMetricAccountID  int64
	SmartRoutingMaxAttempts int
	SmartRoutingMaxDuration time.Duration
	CapacityRetries         int
	// RetryAccountID pins only the next pool-mode retry to the same upstream
	// website. It is request-local and cleared as soon as that website is
	// selected; it is not session stickiness.
	RetryAccountID      int64
	Observer            SmartRoutingAttemptObserver
	ContinuityGroupID   int64
	ContinuityAccountID int64
	ProbeToken          string
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
	// PermissionRejectedGroups records candidates that failed the final
	// request-time authorization check. It is separate from TriedGroups because
	// the current-group retry path intentionally runs before that generic check.
	PermissionRejectedGroups map[int64]struct{}
	EndpointType             string
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
	// SmartRoutingGroupAdapterUnavailable asks the Overlay to confirm whether
	// an enabled route exists before the shared selector can continue without
	// its group repository. The selector clears it after the resolver returns.
	SmartRoutingGroupAdapterUnavailable bool
	// ContinuityLocked is set for requests carrying previous_response_id.
	// Such requests must stay with their verified response owner; moving the
	// response chain to another group would produce an upstream 404/401 or,
	// worse, a semantically different conversation.
	ContinuityLocked bool
	FinalSuccess     bool
	FinalGroupID     int64
	FinalAccountID   int64
	FinalErrorStatus int
	FirstTokenMs     int64
	OutcomeRecorded  bool
}

const (
	// SmartRoutingDefaultMaxAttempts bounds stacked pool-mode retries and
	// cross-account/group failover. A pool website still gets its configured
	// same-site retry budget, while the request cannot fan out indefinitely.
	SmartRoutingDefaultMaxAttempts = 12
	// SmartRoutingDefaultMaxDuration is retained for compatibility with callers
	// that still populate the legacy duration field. It must not become a hard
	// timeout for an in-flight model response; routing fan-out is bounded by
	// send counts instead.
	SmartRoutingDefaultMaxDuration = 30 * time.Second
	// SmartRoutingMaxSiteAttempts bounds real OpenAI sends to one routed
	// website/account. It includes the initial send and all internal retries.
	SmartRoutingMaxSiteAttempts = 2
)

const SmartRoutingAttemptBudgetExceededReason GatewayFailureReason = "smart_routing_attempt_budget_exceeded"
const SmartRoutingTurnAttemptBudgetExceededReason GatewayFailureReason = "smart_routing_turn_attempt_budget_exceeded"
const SmartRoutingSiteAttemptBudgetExceededReason GatewayFailureReason = "smart_routing_site_attempt_budget_exceeded"

func prepareSmartRoutingAttemptBudget(state *SmartRoutingRequestState) {
	if state == nil {
		return
	}
	state.attemptMu.Lock()
	defer state.attemptMu.Unlock()
	if state.SmartRoutingStartedAt.IsZero() {
		state.SmartRoutingStartedAt = time.Now()
	}
	if state.SmartRoutingMaxAttempts <= 0 {
		state.SmartRoutingMaxAttempts = SmartRoutingDefaultMaxAttempts
	}
}

// PrepareSmartRoutingOpenAIRequestBudget starts the request-level timer before
// the first OpenAI send without consuming an attempt slot. Attempts are
// reserved at the transport boundary, after account admission and immediately
// before a real upstream send.
func PrepareSmartRoutingOpenAIRequestBudget(ctx context.Context) {
	state := SmartRoutingStateFromContext(ctx)
	if state == nil || !state.SmartRoutingActive {
		return
	}
	prepareSmartRoutingAttemptBudget(state)
}

func smartRoutingAttemptGroupID(ctx context.Context, state *SmartRoutingRequestState) int64 {
	if state != nil && state.CurrentGroupID > 0 {
		return state.CurrentGroupID
	}
	if state != nil && state.MainGroupID > 0 {
		return state.MainGroupID
	}
	if key, ok := APIKeyFromContext(ctx); ok && key != nil {
		if key.GroupID != nil && *key.GroupID > 0 {
			return *key.GroupID
		}
		if key.Group != nil && key.Group.ID > 0 {
			return key.Group.ID
		}
	}
	return 0
}

func smartRoutingAttemptBudgetError(reason GatewayFailureReason, scope GatewayFailureScope, accountID int64) *UpstreamFailoverError {
	if reason == SmartRoutingSiteAttemptBudgetExceededReason {
		return &UpstreamFailoverError{
			StatusCode:             http.StatusServiceUnavailable,
			RequestScopedTransient: true,
			Reason:                 reason,
			Scope:                  GatewayFailureScopeAccount,
			NextAccountAction:      NextAccountLegacyRetry,
			ClientStatusCode:       http.StatusServiceUnavailable,
			ClientMessage:          "OpenAI upstream attempt limit reached; switching to another upstream",
		}
	}
	return &UpstreamFailoverError{
		StatusCode:        http.StatusServiceUnavailable,
		Reason:            reason,
		Scope:             scope,
		NextAccountAction: NextAccountStop,
		ClientStatusCode:  http.StatusServiceUnavailable,
		ClientMessage:     "smart routing attempt budget exhausted",
	}
}

func reserveSmartRoutingOpenAIAttempt(ctx context.Context, accountID int64, turn bool) error {
	state := SmartRoutingStateFromContext(ctx)
	if state == nil || !state.SmartRoutingActive {
		return nil
	}
	if ctx == nil || ctx.Err() != nil {
		if ctx != nil {
			return ctx.Err()
		}
		return context.Canceled
	}
	if accountID <= 0 {
		return smartRoutingAttemptBudgetError(SmartRoutingAttemptBudgetExceededReason, GatewayFailureScopeRequest, 0)
	}

	state.attemptMu.Lock()
	defer state.attemptMu.Unlock()

	maxAttempts := state.SmartRoutingMaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = SmartRoutingDefaultMaxAttempts
		state.SmartRoutingMaxAttempts = maxAttempts
	}
	now := time.Now()
	if turn {
		if state.turnAttemptCount >= maxAttempts {
			return smartRoutingAttemptBudgetError(SmartRoutingTurnAttemptBudgetExceededReason, GatewayFailureScopeRequest, accountID)
		}
		if state.turnAttemptStart.IsZero() {
			state.turnAttemptStart = now
		}
	} else {
		if state.SmartRoutingStartedAt.IsZero() {
			state.SmartRoutingStartedAt = now
		}
		if state.TotalAttempts >= maxAttempts {
			return smartRoutingAttemptBudgetError(SmartRoutingAttemptBudgetExceededReason, GatewayFailureScopeRequest, accountID)
		}
	}

	groupID := smartRoutingAttemptGroupID(ctx, state)
	if groupID <= 0 {
		return smartRoutingAttemptBudgetError(SmartRoutingAttemptBudgetExceededReason, GatewayFailureScopeRequest, accountID)
	}
	if state.siteAttempts == nil {
		state.siteAttempts = make(map[smartRoutingSiteAttemptKey]int)
	}
	key := smartRoutingSiteAttemptKey{groupID: groupID, accountID: accountID}
	if state.siteAttempts[key] >= SmartRoutingMaxSiteAttempts {
		return smartRoutingAttemptBudgetError(SmartRoutingSiteAttemptBudgetExceededReason, GatewayFailureScopeAccount, accountID)
	}
	state.siteAttempts[key]++
	if turn {
		state.turnAttemptCount++
	} else {
		state.TotalAttempts++
	}
	return nil
}

// ReserveSmartRoutingRequestAttempt accounts one actual upstream request send.
// Its request-wide and (group, account) counters are shared by handler calls,
// protocol repair, and internal transport retries. The implementation is
// platform-neutral even though OpenAI is currently the primary consumer.
func ReserveSmartRoutingRequestAttempt(ctx context.Context, accountID int64) error {
	return reserveSmartRoutingOpenAIAttempt(ctx, accountID, false)
}

// ReserveSmartRoutingOpenAIRequestAttempt is kept as a compatibility alias for
// OpenAI-specific callers. The budget itself is shared by every smart-routed
// platform so a non-OpenAI gateway retry cannot bypass the request cap.
func ReserveSmartRoutingOpenAIRequestAttempt(ctx context.Context, accountID int64) error {
	return ReserveSmartRoutingRequestAttempt(ctx, accountID)
}

// ReserveSmartRoutingOpenAIWSTurnAttempt accounts one actual Responses
// WebSocket turn send. The site counter is shared with every retry for the
// same account and reset only when that logical turn terminates.
func ReserveSmartRoutingOpenAIWSTurnAttempt(ctx context.Context, accountID int64) error {
	return reserveSmartRoutingOpenAIAttempt(ctx, accountID, true)
}

// SmartRoutingAttemptBudgetAvailable reserves one upstream attempt. It is
// intentionally request-local, so concurrent requests never share a budget.
func SmartRoutingAttemptBudgetAvailable(ctx context.Context) bool {
	state := SmartRoutingStateFromContext(ctx)
	if state == nil || !state.SmartRoutingActive || ctx == nil || ctx.Err() != nil {
		return false
	}
	state.attemptMu.Lock()
	defer state.attemptMu.Unlock()
	if state.SmartRoutingStartedAt.IsZero() {
		state.SmartRoutingStartedAt = time.Now()
	}
	if state.SmartRoutingMaxAttempts <= 0 {
		state.SmartRoutingMaxAttempts = SmartRoutingDefaultMaxAttempts
	}
	if state.TotalAttempts >= state.SmartRoutingMaxAttempts {
		return false
	}
	state.TotalAttempts++
	return true
}

// SmartRoutingInternalAttemptBudgetAvailable reserves a request attempt for a
// provider call made inside an already admitted account attempt (for example,
// an OpenAI WS reconnect or a protocol-repair replay). It deliberately shares
// the same counter as the handler-level admission, so internal retries cannot
// multiply the configured smart-routing fan-out. Ordinary single-group
// requests are unaffected.
func SmartRoutingInternalAttemptBudgetAvailable(ctx context.Context) bool {
	state := SmartRoutingStateFromContext(ctx)
	if state == nil || !state.SmartRoutingActive {
		return true
	}
	return SmartRoutingAttemptBudgetAvailable(ctx)
}

// SmartRoutingTurnAttemptBudgetAvailable reserves one upstream request for the
// current logical Responses WebSocket turn. A WebSocket can carry many
// response.create turns over one connection, so its retry budget is isolated
// from the request-level HTTP budget and is cleared only after a successful
// terminal response. Account/group failover re-entry shares this same state.
func SmartRoutingTurnAttemptBudgetAvailable(ctx context.Context) bool {
	state := SmartRoutingStateFromContext(ctx)
	if state == nil || !state.SmartRoutingActive {
		return true
	}
	if ctx == nil || ctx.Err() != nil {
		return false
	}
	state.attemptMu.Lock()
	defer state.attemptMu.Unlock()
	maxAttempts := state.SmartRoutingMaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = SmartRoutingDefaultMaxAttempts
		state.SmartRoutingMaxAttempts = maxAttempts
	}
	if state.turnAttemptCount >= maxAttempts {
		return false
	}
	if state.turnAttemptStart.IsZero() {
		state.turnAttemptStart = time.Now()
	}
	state.turnAttemptCount++
	return true
}

// CompleteSmartRoutingTurnAttemptBudget closes the current WebSocket turn's
// retry window after a terminal response. Failed turns deliberately retain
// their counters so account/group retries cannot reset the budget.
func CompleteSmartRoutingTurnAttemptBudget(ctx context.Context) {
	state := SmartRoutingStateFromContext(ctx)
	if state == nil || !state.SmartRoutingActive {
		return
	}
	state.attemptMu.Lock()
	state.turnAttemptCount = 0
	state.turnAttemptStart = time.Time{}
	state.siteAttempts = nil
	state.attemptMu.Unlock()
}

// SmartRoutingRetryBudgetAllows is used before sleeping for a same-site pool
// retry. It prevents a retry delay from starting after the request budget has
// already expired, while leaving ordinary single-group retry behavior intact.
func SmartRoutingRetryBudgetAllows(ctx context.Context) bool {
	state := SmartRoutingStateFromContext(ctx)
	if state == nil || !state.SmartRoutingActive {
		return true
	}
	state.attemptMu.Lock()
	defer state.attemptMu.Unlock()
	if state.SmartRoutingStartedAt.IsZero() {
		return true
	}
	if state.SmartRoutingMaxAttempts > 0 && state.TotalAttempts >= state.SmartRoutingMaxAttempts {
		return false
	}
	return true
}

// SmartRoutingAttemptContext preserves the caller's request lifetime. Routing
// budgets cap the number of upstream sends, not model-generation time; applying
// a route deadline here would terminate valid long-running streams and prevent
// a slow failed upstream from switching to another candidate.
func SmartRoutingAttemptContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		return context.Background(), func() {}
	}
	return ctx, func() {}
}

func SmartRoutingSuppressSameAccountRetry(ctx context.Context, err *UpstreamFailoverError) bool {
	state := SmartRoutingStateFromContext(ctx)
	if state == nil || !state.SmartRoutingActive || state.Observer == nil || err == nil {
		return false
	}
	policy, ok := state.Observer.(SmartRoutingSameAccountRetryPolicy)
	return ok && policy.SuppressSameAccountRetry(err)
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

func resolveSmartRoutingWithoutGroupRepository(
	ctx context.Context,
	groupID *int64,
	requestedModel, sessionHash string,
	excludedAccountIDs map[int64]struct{},
	resolver SmartRoutingResolver,
) *int64 {
	state := SmartRoutingStateFromContext(ctx)
	if state == nil || resolver == nil {
		return groupID
	}
	if state.SmartRoutingActive {
		MarkSmartRoutingContractInvalid(ctx)
		if apiKey, ok := APIKeyFromContext(ctx); ok && apiKey != nil {
			RestoreSmartRoutingPrimarySnapshot(ctx, apiKey)
		}
		if state.MainGroupID > 0 {
			primaryID := state.MainGroupID
			return &primaryID
		}
		return groupID
	}
	apiKey, ok := APIKeyFromContext(ctx)
	if !ok || apiKey == nil || apiKey.ID <= 0 {
		return groupID
	}

	previous := state.SmartRoutingGroupAdapterUnavailable
	state.SmartRoutingGroupAdapterUnavailable = true
	_, _ = resolver.Resolve(ctx, apiKey, requestedModel, sessionHash, excludedAccountIDs)
	state.SmartRoutingGroupAdapterUnavailable = previous
	if !state.SmartRoutingActive {
		return groupID
	}

	MarkSmartRoutingContractInvalid(ctx)
	RestoreSmartRoutingPrimarySnapshot(ctx, apiKey)
	if state.MainGroupID > 0 {
		primaryID := state.MainGroupID
		return &primaryID
	}
	return groupID
}

// SmartRoutingForwardModelCompatible validates the model that will actually be
// sent to an OpenAI-compatible upstream. This final check is intentionally
// smart-routing-only; legacy single-group channel mappings retain their
// existing behavior.
func SmartRoutingForwardModelCompatible(ctx context.Context, model string) bool {
	state := SmartRoutingStateFromContext(ctx)
	if state == nil || !state.SmartRoutingActive {
		return true
	}
	if knownForeignModelFamilyForModel(model) == "" {
		return true
	}
	MarkSmartRoutingContractInvalid(ctx)
	return false
}

func SetSmartRoutingRetryAccount(ctx context.Context, accountID int64) {
	state := SmartRoutingStateFromContext(ctx)
	if state == nil || !state.SmartRoutingActive || accountID <= 0 {
		return
	}
	state.RetryAccountID = accountID
}

func ClearSmartRoutingRetryAccount(ctx context.Context) {
	if state := SmartRoutingStateFromContext(ctx); state != nil {
		state.RetryAccountID = 0
	}
}

func CompleteSmartRoutingRetrySelection(ctx context.Context, accountID int64) {
	state := SmartRoutingStateFromContext(ctx)
	if state == nil || state.RetryAccountID == 0 {
		return
	}
	if state.RetryAccountID == accountID {
		state.RetryAccountID = 0
	}
}

func SmartRoutingRetryAccountID(ctx context.Context) int64 {
	if state := SmartRoutingStateFromContext(ctx); state != nil && state.SmartRoutingActive {
		return state.RetryAccountID
	}
	return 0
}

// LimitSmartRoutingAccountsForRetry keeps a pool retry on the same upstream
// website. If that website disappeared from the fresh schedulable snapshot,
// the pin is cleared and the ordinary pool remains available.
func LimitSmartRoutingAccountsForRetry(ctx context.Context, accounts []Account) []Account {
	state := SmartRoutingStateFromContext(ctx)
	if state == nil || !state.SmartRoutingActive || state.RetryAccountID <= 0 {
		return accounts
	}
	for i := range accounts {
		if accounts[i].ID == state.RetryAccountID {
			return []Account{accounts[i]}
		}
	}
	state.RetryAccountID = 0
	return accounts
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
	// Empty or future subscription types must not compare equal by accident.
	// The routing contract is deliberately closed over the two billing modes
	// understood by the billing and subscription services.
	if !smartRoutingKnownSubscriptionType(primary.SubscriptionType) || !smartRoutingKnownSubscriptionType(candidate.SubscriptionType) {
		return false
	}
	return primary.SubscriptionType == candidate.SubscriptionType
}

func smartRoutingKnownSubscriptionType(subscriptionType string) bool {
	return subscriptionType == SubscriptionTypeStandard || subscriptionType == SubscriptionTypeSubscription
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
	if !smartRoutingKnownSubscriptionType(current.SubscriptionType) {
		return false
	}
	if snapshot == nil {
		return true
	}
	if snapshot.ID != primaryID || snapshot.Platform == "" || snapshot.Platform == PlatformComposite {
		return false
	}
	if !smartRoutingKnownSubscriptionType(snapshot.SubscriptionType) || !smartRoutingKnownSubscriptionType(current.SubscriptionType) {
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

// MarkSmartRoutingAccountFailed records an account failure without allowing
// the exclusion to leak into another routed group. FailedAccountIDs remains a
// compatibility/audit aggregate; schedulers must use the group-scoped helper
// below when deciding whether an account is unavailable for this request.
func MarkSmartRoutingAccountFailed(ctx context.Context, accountID, groupID int64) {
	state := SmartRoutingStateFromContext(ctx)
	if state == nil || accountID <= 0 {
		return
	}
	if groupID <= 0 {
		groupID = state.CurrentGroupID
	}
	if state.FailedAccountIDs == nil {
		state.FailedAccountIDs = make(map[int64]struct{})
	}
	state.FailedAccountIDs[accountID] = struct{}{}
	if groupID <= 0 {
		return
	}
	if state.FailedAccountGroups == nil {
		state.FailedAccountGroups = make(map[int64]map[int64]struct{})
	}
	groups := state.FailedAccountGroups[accountID]
	if groups == nil {
		groups = make(map[int64]struct{})
		state.FailedAccountGroups[accountID] = groups
	}
	groups[groupID] = struct{}{}
}

// SmartRoutingAccountFailed reports whether this account failed in the
// specified group during the current request. Legacy states created by tests
// or older callers without the scoped map intentionally do not fall back to
// the flat set: treating that set as global would recreate the cross-group
// quarantine bug.
func SmartRoutingAccountFailed(ctx context.Context, accountID, groupID int64) bool {
	state := SmartRoutingStateFromContext(ctx)
	if state == nil || accountID <= 0 || groupID <= 0 || state.FailedAccountGroups == nil {
		return false
	}
	_, failed := state.FailedAccountGroups[accountID][groupID]
	return failed
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
	state.RetryAccountID = 0
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
