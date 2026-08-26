package smartrouting

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const (
	defaultMaxCandidates = 8
	minSamples           = 20
)

type Profile struct {
	APIKeyID        int64   `json:"api_key_id"`
	Enabled         bool    `json:"enabled"`
	Mode            string  `json:"mode"`
	Strategy        string  `json:"strategy"`
	PriceWeight     float64 `json:"price_weight"`
	SpeedWeight     float64 `json:"speed_weight"`
	SuccessWeight   float64 `json:"success_weight"`
	StickyTTL       int     `json:"sticky_ttl_seconds"`
	ExplorationRate float64 `json:"exploration_rate"`
	Version         int64   `json:"version"`
	Groups          []Group `json:"groups"`
}

type Group struct {
	GroupID       int64   `json:"group_id"`
	SortOrder     int     `json:"sort_order"`
	Enabled       bool    `json:"enabled"`
	Name          string  `json:"name,omitempty"`
	Platform      string  `json:"platform,omitempty"`
	Price         float64 `json:"price"`
	SuccessRate   float64 `json:"success_rate"`
	SpeedMs       int64   `json:"speed_ms"`
	Samples       int64   `json:"samples"`
	CapacityRatio float64 `json:"capacity_ratio"`
	Score         float64 `json:"score"`
	Eligible      bool    `json:"eligible"`
	Reason        string  `json:"reason,omitempty"`
}

type UpdateRequest struct {
	Version         int64   `json:"version"`
	Enabled         bool    `json:"enabled"`
	Mode            string  `json:"mode"`
	Strategy        string  `json:"strategy"`
	PriceWeight     float64 `json:"price_weight"`
	SpeedWeight     float64 `json:"speed_weight"`
	SuccessWeight   float64 `json:"success_weight"`
	StickyTTL       int     `json:"sticky_ttl_seconds"`
	ExplorationRate float64 `json:"exploration_rate"`
	GroupIDs        []int64 `json:"group_ids"`
}

type PreviewRequest struct {
	Model    string `json:"model"`
	Endpoint string `json:"endpoint"`
}

type stats struct {
	Samples         int64
	Success         int64
	SpeedMs         sql.NullInt64
	FallbackSamples int64
	FallbackSuccess int64
	FallbackSpeedMs sql.NullInt64
}

func (m *Module) loadStats(ctx context.Context, groupID int64, model string) stats {
	if m == nil || m.repo == nil || m.repo.db == nil {
		return stats{}
	}
	var st stats
	// Keep the realtime and fallback windows in one query. A route with a cold
	// candidate can otherwise issue two DB round trips per group on every first
	// request, multiplying latency exactly when the pool is warming up.
	_ = m.repo.db.QueryRowContext(ctx, `SELECT
		COUNT(*) FILTER (WHERE ul.duration_ms IS NOT NULL AND ul.created_at>=NOW()-INTERVAL '15 minutes'),
		COUNT(*) FILTER (WHERE ul.duration_ms IS NOT NULL AND NOT EXISTS (SELECT 1 FROM ops_error_logs oe WHERE oe.request_id=ul.request_id) AND ul.created_at>=NOW()-INTERVAL '15 minutes'),
		COALESCE(percentile_cont(0.5) WITHIN GROUP (ORDER BY COALESCE(ul.first_token_ms, ul.duration_ms)) FILTER (WHERE ul.duration_ms IS NOT NULL AND ul.created_at>=NOW()-INTERVAL '15 minutes'),0)::bigint,
		COUNT(*) FILTER (WHERE ul.duration_ms IS NOT NULL AND ul.created_at>=NOW()-INTERVAL '1 hour'),
		COUNT(*) FILTER (WHERE ul.duration_ms IS NOT NULL AND NOT EXISTS (SELECT 1 FROM ops_error_logs oe WHERE oe.request_id=ul.request_id) AND ul.created_at>=NOW()-INTERVAL '1 hour'),
		COALESCE(percentile_cont(0.5) WITHIN GROUP (ORDER BY COALESCE(ul.first_token_ms, ul.duration_ms)) FILTER (WHERE ul.duration_ms IS NOT NULL AND ul.created_at>=NOW()-INTERVAL '1 hour'),0)::bigint
		FROM usage_logs ul WHERE ul.group_id=$1 AND ($2='' OR ul.requested_model=$2) AND ul.created_at>=NOW()-INTERVAL '1 hour'`, groupID, model).Scan(&st.Samples, &st.Success, &st.SpeedMs, &st.FallbackSamples, &st.FallbackSuccess, &st.FallbackSpeedMs)
	return st
}

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Get(ctx context.Context, keyID int64) (*Profile, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("smart routing repository unavailable")
	}
	var p Profile
	err := r.db.QueryRowContext(ctx, `SELECT api_key_id, enabled, mode, strategy, price_weight, speed_weight, success_weight, sticky_ttl_seconds, exploration_rate, version FROM custom_api_key_route_profiles WHERE api_key_id=$1`, keyID).
		Scan(&p.APIKeyID, &p.Enabled, &p.Mode, &p.Strategy, &p.PriceWeight, &p.SpeedWeight, &p.SuccessWeight, &p.StickyTTL, &p.ExplorationRate, &p.Version)
	if errors.Is(err, sql.ErrNoRows) {
		// Version 0 is the optimistic-lock sentinel for a profile that does
		// not exist yet. Returning 1 here lets two clients both submit their
		// first edit with the same version and silently overwrite each other.
		return &Profile{APIKeyID: keyID, Mode: "single", Strategy: "auto", PriceWeight: 40, SpeedWeight: 30, SuccessWeight: 30, StickyTTL: 1800, ExplorationRate: 0.05, Version: 0, Groups: []Group{}}, nil
	}
	if err != nil {
		return nil, err
	}
	// Keep the API shape stable for legacy keys and profiles without
	// candidates. JSON null would make clients treat a normal single-group
	// profile as a failed routing load when they call groups.filter(...).
	p.Groups = make([]Group, 0)
	rows, err := r.db.QueryContext(ctx, `SELECT rg.group_id, rg.sort_order, rg.enabled, COALESCE(g.name,''), COALESCE(g.platform,'') FROM custom_api_key_route_groups rg LEFT JOIN groups g ON g.id=rg.group_id WHERE rg.profile_id=$1 ORDER BY rg.sort_order, rg.group_id`, keyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var g Group
		if err := rows.Scan(&g.GroupID, &g.SortOrder, &g.Enabled, &g.Name, &g.Platform); err != nil {
			return nil, err
		}
		p.Groups = append(p.Groups, g)
	}
	return &p, rows.Err()
}

func (r *Repository) Save(ctx context.Context, keyID int64, req UpdateRequest) (*Profile, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("smart routing repository unavailable")
	}
	if keyID <= 0 {
		return nil, fmt.Errorf("invalid api key id")
	}
	if err := validateUpdate(req); err != nil {
		return nil, err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	// A row-level lock cannot protect the no-row case. Serialize saves by key
	// so two first writes cannot both insert version 1 and lose an update.
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1)`, keyID); err != nil {
		return nil, err
	}
	var current int64
	err = tx.QueryRowContext(ctx, `SELECT version FROM custom_api_key_route_profiles WHERE api_key_id=$1 FOR UPDATE`, keyID).Scan(&current)
	exists := err == nil
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if req.Version == 0 {
		if exists {
			return nil, fmt.Errorf("routing profile version conflict")
		}
	} else if !exists || current != req.Version {
		return nil, fmt.Errorf("routing profile version conflict")
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO custom_api_key_route_profiles (api_key_id,enabled,mode,strategy,price_weight,speed_weight,success_weight,sticky_ttl_seconds,exploration_rate,version) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,1) ON CONFLICT (api_key_id) DO UPDATE SET enabled=EXCLUDED.enabled, mode=EXCLUDED.mode, strategy=EXCLUDED.strategy, price_weight=EXCLUDED.price_weight, speed_weight=EXCLUDED.speed_weight, success_weight=EXCLUDED.success_weight, sticky_ttl_seconds=EXCLUDED.sticky_ttl_seconds, exploration_rate=EXCLUDED.exploration_rate, version=custom_api_key_route_profiles.version+1, updated_at=NOW()`, keyID, req.Enabled, req.Mode, req.Strategy, req.PriceWeight, req.SpeedWeight, req.SuccessWeight, req.StickyTTL, req.ExplorationRate)
	if err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM custom_api_key_route_groups WHERE profile_id=$1`, keyID); err != nil {
		return nil, err
	}
	for i, gid := range req.GroupIDs {
		if gid <= 0 {
			return nil, fmt.Errorf("group_ids must be positive")
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO custom_api_key_route_groups(profile_id,group_id,sort_order,enabled) VALUES($1,$2,$3,TRUE)`, keyID, gid, i); err != nil {
			return nil, err
		}
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return r.Get(ctx, keyID)
}

func validateUpdate(req UpdateRequest) error {
	if req.Version < 0 {
		return fmt.Errorf("version must not be negative")
	}
	if req.Mode != "single" && req.Mode != "smart" {
		return fmt.Errorf("mode must be single or smart")
	}
	if req.Strategy != "auto" && req.Strategy != "price" && req.Strategy != "speed" && req.Strategy != "success" && req.Strategy != "custom" {
		return fmt.Errorf("invalid strategy")
	}
	if !finiteNonNegative(req.PriceWeight) || !finiteNonNegative(req.SpeedWeight) || !finiteNonNegative(req.SuccessWeight) || math.Abs(req.PriceWeight+req.SpeedWeight+req.SuccessWeight-100) > 0.001 {
		return fmt.Errorf("weights must be non-negative and sum to 100")
	}
	if req.StickyTTL < 0 || req.StickyTTL > 86400 {
		return fmt.Errorf("sticky_ttl_seconds out of range")
	}
	if !finiteNonNegative(req.ExplorationRate) || req.ExplorationRate > 0.05 {
		return fmt.Errorf("exploration_rate must be between 0 and 0.05")
	}
	if req.Mode == "smart" && len(req.GroupIDs) == 0 {
		return fmt.Errorf("smart mode requires at least one group")
	}
	if len(req.GroupIDs) > defaultMaxCandidates {
		return fmt.Errorf("too many candidate groups")
	}
	seen := make(map[int64]struct{}, len(req.GroupIDs))
	for _, groupID := range req.GroupIDs {
		if groupID <= 0 {
			return fmt.Errorf("group_ids must be positive")
		}
		if _, exists := seen[groupID]; exists {
			return fmt.Errorf("duplicate candidate group %d", groupID)
		}
		seen[groupID] = struct{}{}
	}
	return nil
}

type availableGroupsChecker func(context.Context, int64) ([]service.Group, error)

// routingProfileUsable guards the read path as well as validateUpdate. A
// profile can predate current validation or be changed outside the API; bad
// values must never reach score sorting or Redis TTL arithmetic. Returning
// false makes Resolve fail open to the authenticated primary group.
func routingProfileUsable(p *Profile) bool {
	if p == nil || p.Mode != "smart" {
		return false
	}
	if p.Strategy != "auto" && p.Strategy != "price" && p.Strategy != "speed" && p.Strategy != "success" && p.Strategy != "custom" {
		return false
	}
	if !finiteNonNegative(p.PriceWeight) || !finiteNonNegative(p.SpeedWeight) || !finiteNonNegative(p.SuccessWeight) || math.Abs(p.PriceWeight+p.SpeedWeight+p.SuccessWeight-100) > 0.001 {
		return false
	}
	if p.StickyTTL < 0 || p.StickyTTL > 86400 || !finiteNonNegative(p.ExplorationRate) || p.ExplorationRate > 0.05 {
		return false
	}
	if len(p.Groups) == 0 || len(p.Groups) > defaultMaxCandidates {
		return false
	}
	seen := make(map[int64]struct{}, len(p.Groups))
	for _, group := range p.Groups {
		if group.GroupID <= 0 {
			return false
		}
		if _, exists := seen[group.GroupID]; exists {
			return false
		}
		seen[group.GroupID] = struct{}{}
	}
	return true
}

type Module struct {
	repo    *Repository
	apiKeys *service.APIKeyService
	// availableGroups is kept as a narrow dependency so candidate permission
	// checks fail closed when the API-key service is unavailable. NewModule
	// wires the real service; tests can provide an explicit read-only stub.
	availableGroups availableGroupsChecker
	groups          service.GroupRepository
	accounts        service.AccountRepository
	gateway         *service.GatewayService
	openAI          *service.OpenAIGatewayService
	redis           *redis.Client
}

func NewModule(db *sql.DB, apiKeys *service.APIKeyService, groups service.GroupRepository, accounts service.AccountRepository, gateway *service.GatewayService, openAI *service.OpenAIGatewayService, redisClient *redis.Client) *Module {
	m := &Module{repo: NewRepository(db), apiKeys: apiKeys, groups: groups, accounts: accounts, gateway: gateway, openAI: openAI, redis: redisClient}
	if apiKeys != nil {
		m.availableGroups = apiKeys.GetAvailableGroups
	}
	if gateway != nil {
		gateway.SetSmartRoutingResolver(m)
	}
	if openAI != nil {
		openAI.SetSmartRoutingResolver(m)
	}
	return m
}

func (m *Module) RegisterRoutes(authenticated *gin.RouterGroup) {
	if m == nil || authenticated == nil {
		return
	}
	keys := authenticated.Group("/keys/:id/routing")
	keys.GET("", m.get)
	keys.PUT("", m.update)
	keys.POST("/preview", m.preview)
}

func (m *Module) ownedKey(c *gin.Context) (*service.APIKey, bool) {
	if m == nil || m.apiKeys == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"detail": "smart routing unavailable"})
		return nil, false
	}
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"detail": "User not authenticated"})
		return nil, false
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid key ID"})
		return nil, false
	}
	key, err := m.apiKeys.GetByID(c.Request.Context(), id)
	if err != nil || key == nil || key.UserID != subject.UserID {
		c.JSON(http.StatusNotFound, gin.H{"detail": "API key not found"})
		return nil, false
	}
	return key, true
}

func (m *Module) get(c *gin.Context) {
	key, ok := m.ownedKey(c)
	if !ok {
		return
	}
	p, err := m.repo.Get(c.Request.Context(), key.ID)
	if err != nil {
		c.JSON(500, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(200, p)
}

func (m *Module) update(c *gin.Context) {
	key, ok := m.ownedKey(c)
	if !ok {
		return
	}
	var req UpdateRequest
	if c.ShouldBindJSON(&req) != nil {
		c.JSON(400, gin.H{"detail": "Invalid request"})
		return
	}
	if req.Mode == "" {
		req.Mode = "single"
	}
	if req.Strategy == "" {
		req.Strategy = "auto"
	}
	if req.PriceWeight == 0 && req.SpeedWeight == 0 && req.SuccessWeight == 0 {
		req.PriceWeight, req.SpeedWeight, req.SuccessWeight = 40, 30, 30
	}
	if err := m.validateGroups(c, key, &req); err != nil {
		c.JSON(400, gin.H{"detail": err.Error()})
		return
	}
	p, err := m.repo.Save(c.Request.Context(), key.ID, req)
	if err != nil {
		c.JSON(400, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(200, p)
}

func (m *Module) validateGroups(ctx *gin.Context, key *service.APIKey, req *UpdateRequest) error {
	if req.Mode != "smart" {
		req.GroupIDs = nil
		return nil
	}
	if key.GroupID == nil || key.Group == nil {
		return fmt.Errorf("primary group is required")
	}
	checkAvailableGroups := m.availableGroups
	if checkAvailableGroups == nil && m.apiKeys != nil {
		checkAvailableGroups = m.apiKeys.GetAvailableGroups
	}
	if checkAvailableGroups == nil {
		return fmt.Errorf("group permission service unavailable")
	}
	allowed, err := checkAvailableGroups(ctx.Request.Context(), key.UserID)
	if err != nil {
		return err
	}
	allowedMap := map[int64]service.Group{}
	for _, g := range allowed {
		allowedMap[g.ID] = g
	}
	main := key.Group
	for _, gid := range req.GroupIDs {
		if gid == main.ID {
			return fmt.Errorf("group %d must differ from primary group", gid)
		}
		g, ok := allowedMap[gid]
		if !ok {
			return fmt.Errorf("group %d is not available", gid)
		}
		if g.Platform != main.Platform || g.SubscriptionType != main.SubscriptionType || g.Platform == service.PlatformComposite {
			return fmt.Errorf("group %d is incompatible with primary group", gid)
		}
	}
	return nil
}

func (m *Module) preview(c *gin.Context) {
	key, ok := m.ownedKey(c)
	if !ok {
		return
	}
	var req PreviewRequest
	_ = c.ShouldBindJSON(&req)
	req.Model = strings.TrimSpace(req.Model)
	req.Endpoint = strings.TrimSpace(req.Endpoint)
	if req.Model == "" {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "model is required"})
		return
	}
	endpoint := normalizeRoutingEndpoint(req.Endpoint)
	if req.Endpoint != "" && endpoint == "" {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "unsupported routing endpoint"})
		return
	}
	if endpoint == "" {
		endpoint = "chat_completions"
	}
	p, err := m.repo.Get(c.Request.Context(), key.ID)
	if err != nil {
		c.JSON(500, gin.H{"detail": err.Error()})
		return
	}
	previewCtx := c.Request.Context()
	if endpoint != "" {
		state := &service.SmartRoutingRequestState{EndpointType: endpoint}
		if key.GroupID != nil {
			state.MainGroupID, state.CurrentGroupID = *key.GroupID, *key.GroupID
		}
		previewCtx = service.WithSmartRoutingState(previewCtx, state)
	}
	ranked, err := m.rank(previewCtx, key, p, req.Model, map[int64]struct{}{})
	if err != nil {
		c.JSON(400, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(200, gin.H{"profile": p, "groups": ranked})
}

func (m *Module) Resolve(ctx context.Context, key *service.APIKey, model, sessionHash string, excluded map[int64]struct{}) (*service.SmartRouteDecision, error) {
	// Clear the per-request marker before any validation or dependency check.
	// A resolver error (including an unavailable repository or malformed key)
	// must fail open to the authenticated primary group and must not leave the
	// scheduler eligible to re-enter the smart-routing retry loop.
	var state *service.SmartRoutingRequestState
	if ctx != nil {
		state = service.SmartRoutingStateFromContext(ctx)
	}
	if state != nil {
		state.SmartRoutingActive = false
	}
	if m == nil || m.repo == nil {
		return nil, errors.New("smart routing repository unavailable")
	}
	if key == nil || key.GroupID == nil {
		return nil, nil
	}
	p, err := m.repo.Get(ctx, key.ID)
	if err != nil || p == nil || !p.Enabled || p.Mode != "smart" {
		return nil, err
	}
	if !routingProfileUsable(p) {
		// Treat malformed persisted configuration as disabled for this request.
		// The authenticated primary group remains the only safe fallback.
		return nil, nil
	}
	if state == nil {
		return nil, nil
	}
	// MVP routing is intentionally limited to OpenAI and Anthropic text
	// contracts. Same-platform validation alone would otherwise let Gemini,
	// Grok, or future platforms inherit a failover path whose billing and
	// session semantics have not been audited yet.
	primary := key.Group
	if state.MainGroupID > 0 && (primary == nil || primary.ID != state.MainGroupID) {
		if m.groups == nil {
			return nil, nil
		}
		resolvedPrimary, groupErr := m.groups.GetByIDLite(ctx, state.MainGroupID)
		if groupErr != nil || resolvedPrimary == nil {
			return nil, nil
		}
		primary = resolvedPrimary
	}
	if !smartRoutingPrimaryPlatformAllowed(primary, state.EndpointType) {
		return nil, nil
	}
	if state.EndpointType != "chat_completions" && state.EndpointType != "responses" && state.EndpointType != "messages" {
		return nil, nil
	}
	// Explicit platform routes (currently /antigravity/...) intentionally
	// override the normal group scheduler. Their account pool and billing
	// semantics are different from the text HTTP contract, so an Anthropic or
	// OpenAI key must not be smart-routed into that forced transport.
	if smartRoutingForcePlatform(ctx) {
		return nil, nil
	}
	// Image generation has a different capability and billing contract. MVP
	// smart routing is text-only; keep these requests on the authenticated
	// primary group until media routing has its own price/capability model.
	if state.ImageGenerationIntent {
		return nil, nil
	}
	// WebSocket and other explicitly transport-specific routes have
	// connection/session semantics that the text HTTP failover contract does
	// not cover. `Any` is the normal HTTP scheduler input for Chat/Responses/
	// Messages and must remain routable; only an explicit WebSocket transport
	// is excluded here.
	if !smartRoutingTransportAllowed(state.RequiredOpenAITransport) {
		return nil, nil
	}
	state.SmartRoutingActive = true
	if mapped := service.OpenAIForwardModelFromContext(ctx); mapped != "" {
		model = mapped
	}
	if state.TriedGroups == nil {
		state.TriedGroups = make(map[int64]struct{})
	}
	maxSwitches := state.MaxGroupSwitches
	if maxSwitches <= 0 {
		maxSwitches = 2
	}
	ranked, err := m.rank(ctx, key, p, model, excluded)
	if err != nil {
		// A routing preflight failure is a fail-open degradation to the
		// authenticated primary group, not an active smart-routing request.
		// Clear the marker before returning so handlers do not re-enter their
		// cross-group retry loop after a primary-group failure.
		state.SmartRoutingActive = false
		return nil, err
	}
	// `rank` deliberately returns diagnostic rows for ineligible candidates so
	// the preview endpoint can explain why they were filtered. They are not a
	// route, however. Clear the active marker before returning nil when the
	// complete candidate pool is unavailable; otherwise the scheduler can
	// treat the restored primary as an active smart route and re-enter its
	// retry loop indefinitely.
	hasEligible := false
	for _, candidate := range ranked {
		if candidate.Eligible {
			hasEligible = true
			break
		}
	}
	if !hasEligible {
		state.SmartRoutingActive = false
		return nil, nil
	}
	stickyID := m.getStickyGroup(ctx, key.ID, p.Version, sessionHash)
	if stickyID > 0 {
		for i := range ranked {
			if ranked[i].GroupID == stickyID && ranked[i].Eligible {
				ranked[0], ranked[i] = ranked[i], ranked[0]
				break
			}
		}
	} else if exploreIndex := m.explorationIndex(key, p, ranked, state.MainGroupID, sessionHash, model); exploreIndex >= 0 {
		// Exploration is applied only after hard eligibility and price checks,
		// and never overrides an existing sticky binding. This gives a cold
		// candidate a bounded opportunity without making routing nondeterministic
		// for a given key/session.
		ranked[0], ranked[exploreIndex] = ranked[exploreIndex], ranked[0]
	}
	for _, g := range ranked {
		if !g.Eligible {
			continue
		}
		if _, rejected := state.BillingRejectedGroups[g.GroupID]; rejected {
			continue
		}
		if !routeStateAllowsGroup(state, g.GroupID, stickyID, maxSwitches) {
			// previous_response_id chains cannot be replayed against another
			// group. Let the normal selector return its explicit no-account
			// error instead of silently changing the conversation owner.
			continue
		}
		// Account failover is intentionally handled inside the current group
		// before moving to another group. The resolver marks a group as tried
		// when it is admitted, so this check must run before the TriedGroups
		// filter on subsequent scheduler calls.
		if g.GroupID == state.CurrentGroupID && len(excluded) > 0 {
			platform := ""
			if key.Group != nil {
				platform = key.Group.Platform
			}
			if m.accounts == nil {
				return nil, nil
			}
			remaining, known := m.remainingModelAccountsKnown(ctx, state.CurrentGroupID, platform, m.dispatchModelForGroup(ctx, key.Group, model), excluded)
			if !known {
				// Preserve the primary route when account discovery is unavailable;
				// an infrastructure error is not evidence that the group is empty.
				return nil, nil
			}
			if remaining <= 0 {
				state.TriedGroups[g.GroupID] = struct{}{}
				continue
			}
			decision := &service.SmartRouteDecision{GroupID: g.GroupID, FallbackDepth: len(state.TriedGroups), Reason: g.Reason}
			m.recordDecision(ctx, key, p, model, g, decision)
			return decision, nil
		}
		if _, tried := state.TriedGroups[g.GroupID]; tried {
			continue
		}
		depth := len(state.TriedGroups)
		decision := &service.SmartRouteDecision{GroupID: g.GroupID, FallbackDepth: depth, Reason: g.Reason}
		m.recordDecision(ctx, key, p, model, g, decision)
		return decision, nil
	}
	// Every eligible row may already be tried, rejected by continuity, or
	// beyond the switch budget. There is no next route for this invocation;
	// disable the marker so the caller falls back once to the primary group
	// rather than spinning through the same resolver path.
	state.SmartRoutingActive = false
	return nil, nil
}

func smartRoutingPrimaryPlatformAllowed(group *service.Group, endpoint string) bool {
	if group == nil || (group.Platform != service.PlatformOpenAI && group.Platform != service.PlatformAnthropic) {
		return false
	}
	return group.Platform != service.PlatformAnthropic || endpoint == "messages"
}

func smartRoutingForcePlatform(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	forcePlatform, ok := ctx.Value(ctxkey.ForcePlatform).(string)
	return ok && strings.TrimSpace(forcePlatform) != ""
}

// CommitSticky records a group only after the gateway has passed billing and
// account admission. Keeping this separate from Resolve prevents an
// ineligible candidate from poisoning the session's sticky binding.
func (m *Module) CommitSticky(ctx context.Context, apiKeyID, groupID int64, sessionHash string) {
	if m == nil || m.repo == nil || apiKeyID <= 0 || groupID <= 0 || sessionHash == "" {
		return
	}
	p, err := m.repo.Get(ctx, apiKeyID)
	if err != nil || p == nil || !p.Enabled || p.Mode != "smart" || !routingProfileUsable(p) {
		return
	}
	m.setStickyGroup(ctx, apiKeyID, p.Version, sessionHash, groupID, time.Duration(p.StickyTTL)*time.Second)
}

func routeStateAllowsGroup(state *service.SmartRoutingRequestState, groupID, stickyID int64, maxSwitches int) bool {
	if state == nil {
		return true
	}
	if state.ContinuityLocked {
		continuityGroupID := state.MainGroupID
		if stickyID > 0 {
			continuityGroupID = stickyID
		}
		if groupID != continuityGroupID {
			return false
		}
	}
	// The primary group is the documented safe-degradation target. Returning
	// to it after candidate exhaustion is not another candidate switch, so the
	// switch budget must not block this path. Continuity-locked requests have
	// already been constrained to their continuity group above.
	if groupID == state.MainGroupID {
		return true
	}
	return state.CurrentGroupID <= 0 || groupID == state.CurrentGroupID || state.GroupSwitches < maxSwitches
}

// smartRoutingTransportAllowed reports whether the request uses the text HTTP
// contract supported by MVP cross-group failover. The scheduler uses Any for
// ordinary HTTP requests because the concrete account transport is selected
// later, so Any must be accepted alongside the explicit HTTPSSE transport.
func smartRoutingTransportAllowed(transport service.OpenAIUpstreamTransport) bool {
	return transport == "" ||
		transport == service.OpenAIUpstreamTransportAny ||
		transport == service.OpenAIUpstreamTransportHTTPSSE
}

// recordDecision is deliberately asynchronous and best-effort. Routing
// telemetry must never add latency or turn a healthy request into a 5xx when
// the audit table is unavailable.
func (m *Module) recordDecision(ctx context.Context, key *service.APIKey, p *Profile, model string, group Group, decision *service.SmartRouteDecision) {
	if m == nil || m.repo == nil || m.repo.db == nil || key == nil || p == nil || decision == nil {
		return
	}
	requestID, _ := ctx.Value(ctxkey.RequestID).(string)
	if requestID == "" {
		return
	}
	snapshot, _ := json.Marshal(map[string]any{"price": group.Price, "success_rate": group.SuccessRate, "speed_ms": group.SpeedMs, "score": group.Score, "samples": group.Samples})
	endpoint := ""
	if state := service.SmartRoutingStateFromContext(ctx); state != nil {
		endpoint = state.EndpointType
	}
	go func() {
		writeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_, _ = m.repo.db.ExecContext(writeCtx, `INSERT INTO custom_route_decisions (request_id,api_key_id,profile_id,requested_model,endpoint_type,selected_group_id,strategy,score_snapshot,fallback_depth,decision_reason) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, requestID, key.ID, p.APIKeyID, model, endpoint, group.GroupID, p.Strategy, snapshot, decision.FallbackDepth, decision.Reason)
	}()
}

// RecordSelectedAccount fills the account that actually passed scheduler
// admission. Group scoring happens before account selection, so the insert
// and this update are intentionally decoupled. A short retry window covers
// the normal race where the asynchronous insert has not committed yet.
func (m *Module) RecordSelectedAccount(ctx context.Context, apiKeyID, groupID, accountID int64) {
	if m == nil || m.repo == nil || m.repo.db == nil || apiKeyID <= 0 || groupID <= 0 || accountID <= 0 {
		return
	}
	requestID, _ := ctx.Value(ctxkey.RequestID).(string)
	if requestID == "" {
		return
	}
	go func() {
		writeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		for attempt := 0; attempt < 5; attempt++ {
			result, err := m.repo.db.ExecContext(writeCtx, `UPDATE custom_route_decisions SET selected_account_id=$1 WHERE id=(SELECT id FROM custom_route_decisions WHERE request_id=$2 AND api_key_id=$3 AND selected_group_id=$4 AND selected_account_id IS NULL ORDER BY created_at DESC, id DESC LIMIT 1)`, accountID, requestID, apiKeyID, groupID)
			if err == nil {
				if rows, rowsErr := result.RowsAffected(); rowsErr == nil && rows > 0 {
					return
				}
			}
			if attempt < 4 {
				time.Sleep(20 * time.Millisecond)
			}
		}
	}()
}

func (m *Module) stickyKey(apiKeyID, version int64, sessionHash string) string {
	digest := sha256.Sum256([]byte(sessionHash))
	return fmt.Sprintf("custom:route:sticky:apikey:%d:v%d:%x", apiKeyID, version, digest[:])
}

func (m *Module) getStickyGroup(ctx context.Context, apiKeyID, version int64, sessionHash string) int64 {
	if m.redis == nil || apiKeyID <= 0 || version <= 0 || sessionHash == "" {
		return 0
	}
	value, err := m.redis.Get(ctx, m.stickyKey(apiKeyID, version, sessionHash)).Result()
	if err != nil {
		return 0
	}
	id, _ := strconv.ParseInt(value, 10, 64)
	return id
}

func (m *Module) setStickyGroup(ctx context.Context, apiKeyID, version int64, sessionHash string, groupID int64, ttl time.Duration) {
	if m.redis == nil || apiKeyID <= 0 || version <= 0 || sessionHash == "" || groupID <= 0 || ttl <= 0 {
		return
	}
	_ = m.redis.Set(ctx, m.stickyKey(apiKeyID, version, sessionHash), strconv.FormatInt(groupID, 10), ttl).Err()
}

func (m *Module) remainingModelAccounts(ctx context.Context, groupID int64, platform, model string, excluded map[int64]struct{}) int {
	remaining, _ := m.remainingModelAccountsKnown(ctx, groupID, platform, model, excluded)
	return remaining
}

// remainingModelAccountsKnown returns whether the repository response was
// authoritative. A repository failure must not be interpreted as "zero
// accounts" because that would route a request away from the authenticated
// primary group during an infrastructure incident. Only a successful query
// proving zero eligible accounts may make a group ineligible.
func (m *Module) remainingModelAccountsKnown(ctx context.Context, groupID int64, platform, model string, excluded map[int64]struct{}) (int, bool) {
	if m == nil || m.accounts == nil {
		return 0, false
	}
	useMixedScheduling := platform == service.PlatformAnthropic || platform == service.PlatformGemini
	var accounts []service.Account
	var err error
	if useMixedScheduling {
		// GatewayService.selectAccountWithMixedScheduling uses the native
		// platform plus mixed-scheduling Antigravity accounts. The routing
		// preflight must inspect the same pool or it can incorrectly exclude a
		// healthy candidate before the scheduler gets a chance to select it.
		accounts, err = m.accounts.ListSchedulableByGroupIDAndPlatforms(ctx, groupID, []string{platform, service.PlatformAntigravity})
	} else {
		accounts, err = m.accounts.ListSchedulableByGroupIDAndPlatform(ctx, groupID, platform)
	}
	if err != nil {
		return 0, false
	}
	if m.groups == nil {
		return 0, false
	}
	group, groupErr := m.groups.GetByIDLite(ctx, groupID)
	if groupErr != nil || group == nil {
		return 0, false
	}
	// Channel model aliases are only meaningful for the OpenAI-compatible
	// Messages dispatcher. Applying them to native Anthropic groups could
	// reject an otherwise valid account because its allow-list is evaluated
	// against an unrelated OpenAI alias.
	if state := service.SmartRoutingStateFromContext(ctx); state != nil &&
		state.EndpointType == "messages" &&
		(group.Platform == service.PlatformOpenAI || group.Platform == service.PlatformGrok) {
		model = m.resolveCandidateModel(ctx, groupID, model)
	}
	if !candidateEndpointAllowed(ctx, group) {
		return 0, true
	}
	remaining := 0
	for i := range accounts {
		account := &accounts[i]
		if useMixedScheduling && account.Platform == service.PlatformAntigravity && !account.IsMixedSchedulingEnabled() {
			continue
		}
		if _, skip := excluded[account.ID]; skip {
			continue
		}
		if model != "" && (!account.IsModelSupported(model) || !account.IsSchedulableForModelWithContext(ctx, model)) {
			continue
		}
		if state := service.SmartRoutingStateFromContext(ctx); state != nil &&
			(state.SmartRoutingActive || state.EndpointType != "") &&
			(account.Platform == service.PlatformOpenAI || account.Platform == service.PlatformGrok) &&
			!service.SmartRoutingAccountModelCompatible(account, model) {
			continue
		}
		if !account.IsSchedulable() {
			continue
		}
		if group.RequireOAuthOnly && !account.IsOAuth() {
			continue
		}
		if group.RequirePrivacySet && !account.IsPrivacySet() {
			continue
		}
		state := service.SmartRoutingStateFromContext(ctx)
		if state != nil {
			if state.RequiredOpenAICapability != "" && !account.SupportsOpenAIEndpointCapability(state.RequiredOpenAICapability) {
				continue
			}
			if state.RequiredOpenAIImageCapability != "" && !account.SupportsOpenAIImageCapability(state.RequiredOpenAIImageCapability) {
				continue
			}
			if state.RequireCompact && !account.AllowsOpenAICompact() {
				continue
			}
		}
		remaining++
	}
	return remaining, true
}

// resolveCandidateModel keeps account eligibility checks aligned with the
// model that the selected group's channel mapping will actually forward. A
// client-facing alias may be absent from an account whitelist while its
// mapped upstream model is explicitly allowed. If no channel mapping is
// available, both gateway adapters return the original model unchanged.
func (m *Module) resolveCandidateModel(ctx context.Context, groupID int64, model string) string {
	if m == nil || strings.TrimSpace(model) == "" {
		return model
	}
	gid := groupID
	if m.openAI != nil {
		mapping, _ := m.openAI.ResolveChannelMappingAndRestrict(ctx, &gid, model)
		if mapping.Mapped && strings.TrimSpace(mapping.MappedModel) != "" {
			return strings.TrimSpace(mapping.MappedModel)
		}
	}
	if m.gateway != nil {
		mapping, _ := m.gateway.ResolveChannelMappingAndRestrict(ctx, &gid, model)
		if mapping.Mapped && strings.TrimSpace(mapping.MappedModel) != "" {
			return strings.TrimSpace(mapping.MappedModel)
		}
	}
	return model
}

func (m *Module) rank(ctx context.Context, key *service.APIKey, p *Profile, model string, excluded map[int64]struct{}) ([]Group, error) {
	if m == nil || m.groups == nil {
		return nil, errors.New("smart routing group repository unavailable")
	}
	if key == nil || p == nil {
		return nil, errors.New("smart routing key or profile is nil")
	}
	main := key.Group
	if state := service.SmartRoutingStateFromContext(ctx); state != nil && state.MainGroupID > 0 {
		main, _ = m.groups.GetByIDLite(ctx, state.MainGroupID)
	}
	if main == nil && key.GroupID != nil {
		main, _ = m.groups.GetByIDLite(ctx, *key.GroupID)
	}
	if main == nil {
		return nil, fmt.Errorf("primary group unavailable")
	}
	allowed := map[int64]bool{}
	permissionServiceUnavailable := false
	checkAvailableGroups := m.availableGroups
	if checkAvailableGroups == nil && m.apiKeys != nil {
		// Preserve compatibility for in-package construction while keeping the
		// production path fail-closed when neither dependency is wired.
		checkAvailableGroups = m.apiKeys.GetAvailableGroups
	}
	if checkAvailableGroups == nil {
		permissionServiceUnavailable = true
	} else {
		available, err := checkAvailableGroups(ctx, key.UserID)
		if err != nil {
			return nil, fmt.Errorf("refresh routing group permissions: %w", err)
		}
		for _, group := range available {
			allowed[group.ID] = true
		}
	}
	now := service.SmartRoutingPricingAtFromContext(ctx)
	userRates := map[int64]float64{}
	if m.apiKeys != nil {
		rates, err := m.apiKeys.GetUserGroupRates(ctx, key.UserID)
		if err != nil {
			return nil, fmt.Errorf("refresh routing group rates: %w", err)
		}
		userRates = rates
	}
	effectivePrice := func(group *service.Group) (float64, bool) {
		if group == nil {
			return 0, false
		}
		multiplier := group.RateMultiplier
		if override, ok := userRates[group.ID]; ok {
			multiplier = override
		}
		// PeakMultiplierAt intentionally falls back to 1.0 for malformed
		// persisted settings so ordinary billing remains available. Routing
		// price protection must be stricter: a corrupt enabled peak multiplier
		// is not a comparable price and must exclude the group.
		if group.PeakRateEnabled && !finiteNonNegative(group.PeakRateMultiplier) {
			return 0, false
		}
		peak := group.PeakMultiplierAt(now)
		price := multiplier * peak
		if !finiteNonNegative(multiplier) || !finiteNonNegative(peak) || !finiteNonNegative(price) {
			return 0, false
		}
		return price, true
	}
	mainModel := m.dispatchModelForGroup(ctx, main, model)
	mainEffectivePrice, mainPriceValid := effectivePrice(main)
	// The primary group is implicit and does not have a persisted route-group
	// sort order. Keep it ahead of the first configured candidate on score ties;
	// configured candidates still follow their drag order starting at zero.
	mainCandidate := Group{GroupID: main.ID, SortOrder: -1, Name: main.Name, Platform: main.Platform, Price: mainEffectivePrice, Eligible: true}
	if !mainPriceValid {
		mainCandidate.Eligible = false
		mainCandidate.Reason = "primary group price unavailable"
	} else if main.Status != service.StatusActive || main.Platform == service.PlatformComposite {
		mainCandidate.Eligible = false
		mainCandidate.Reason = "primary group unavailable"
	} else if !candidateEndpointAllowed(ctx, main) {
		mainCandidate.Eligible = false
		mainCandidate.Reason = "endpoint not supported by group"
	} else if m.accounts != nil {
		remaining, known := m.remainingModelAccountsKnown(ctx, main.ID, main.Platform, mainModel, excluded)
		if known && remaining == 0 {
			mainCandidate.Eligible = false
			mainCandidate.Reason = "no schedulable accounts"
		}
	}
	if mainPriceValid && !m.pricingComparableForGroups(ctx, main, main, model) {
		mainCandidate.Eligible = false
		mainCandidate.Reason = "channel model pricing requires explicit comparison"
		mainPriceValid = false
	}
	candidates := append([]Group{mainCandidate}, p.Groups...)
	seen := map[int64]bool{}
	out := make([]Group, 0, len(candidates))
	for _, g := range candidates {
		if seen[g.GroupID] {
			continue
		}
		seen[g.GroupID] = true
		if g.GroupID != main.ID {
			if !g.Enabled {
				g.Reason = "candidate disabled"
				out = append(out, g)
				continue
			}
			full, err := m.groups.GetByIDLite(ctx, g.GroupID)
			if err != nil || full == nil {
				g.Reason = "group not found"
				out = append(out, g)
				continue
			}
			var candidatePriceValid bool
			g.Name, g.Platform, g.Price = full.Name, full.Platform, 0
			g.Price, candidatePriceValid = effectivePrice(full)
			if !candidatePriceValid || !mainPriceValid {
				g.Reason = "price unavailable for comparison"
				out = append(out, g)
				continue
			}
			if full.Status != service.StatusActive || full.Platform != main.Platform || full.SubscriptionType != main.SubscriptionType || full.Platform == service.PlatformComposite {
				g.Reason = "incompatible group"
				out = append(out, g)
				continue
			}
			if comparable, reason := m.longContextPricingComparable(ctx, main, full); !comparable {
				g.Reason = reason
				out = append(out, g)
				continue
			}
			if !candidateEndpointAllowed(ctx, full) {
				g.Reason = "endpoint not supported by group"
				out = append(out, g)
				continue
			}
			candidateModel := m.dispatchModelForGroup(ctx, full, model)
			if !m.pricingComparableForGroups(ctx, main, full, model) || (candidateModel != model && !m.pricingComparableForGroups(ctx, main, full, candidateModel)) {
				g.Reason = "channel model pricing requires explicit comparison"
				out = append(out, g)
				continue
			}
			if permissionServiceUnavailable {
				g.Reason = "permission service unavailable"
				out = append(out, g)
				continue
			}
			if !allowed[g.GroupID] {
				g.Reason = "permission or subscription unavailable"
				out = append(out, g)
				continue
			}
			if len(main.ModelPricing) > 0 || len(full.ModelPricing) > 0 {
				g.Reason = "model-specific pricing requires explicit comparison"
				out = append(out, g)
				continue
			}
			if g.Price > mainEffectivePrice+1e-9 {
				g.Reason = "price exceeds primary group"
				out = append(out, g)
				continue
			}
			if m.accounts == nil {
				g.Reason = "account scheduler unavailable"
				out = append(out, g)
				continue
			}
			remaining, known := m.remainingModelAccountsKnown(ctx, g.GroupID, main.Platform, candidateModel, excluded)
			if !known {
				g.Reason = "account availability unavailable"
				out = append(out, g)
				continue
			}
			if remaining == 0 {
				g.Reason = "no schedulable accounts"
				out = append(out, g)
				continue
			}
			// Every hard eligibility check passed. Mark the candidate before
			// collecting metrics so rank and Resolve can consider it.
			g.Eligible = true
		}
		st := m.loadStats(ctx, g.GroupID, model)
		if st.Samples < minSamples && st.FallbackSamples > st.Samples {
			st.Samples = st.FallbackSamples
			st.Success = st.FallbackSuccess
			st.SpeedMs = st.FallbackSpeedMs
		}
		g.Samples = st.Samples
		g.SuccessRate = 0.5
		if st.Samples > 0 {
			g.SuccessRate = float64(st.Success+1) / float64(st.Samples+2)
		}
		if st.SpeedMs.Valid {
			g.SpeedMs = st.SpeedMs.Int64
		}
		out = append(out, g)
	}
	eligible := make([]*Group, 0, len(out))
	for i := range out {
		if out[i].Eligible {
			eligible = append(eligible, &out[i])
		}
	}
	if len(eligible) == 0 {
		// Keep preview and diagnostics deterministic even when every group is
		// currently unavailable. Resolve will ignore these entries, but callers
		// still rely on the persisted order when explaining exclusions.
		sortRankedGroups(out)
		return out, nil
	}
	minP, maxP := eligible[0].Price, eligible[0].Price
	maxS, minS := eligible[0].SuccessRate, eligible[0].SuccessRate
	for _, g := range eligible {
		if g.Price < minP {
			minP = g.Price
		}
		if g.Price > maxP {
			maxP = g.Price
		}
		if g.SuccessRate > maxS {
			maxS = g.SuccessRate
		}
		if g.SuccessRate < minS {
			minS = g.SuccessRate
		}
	}
	for _, g := range eligible {
		priceScore := 100.0
		if maxP > minP {
			priceScore = 100 * (maxP - g.Price) / (maxP - minP)
		}
		successScore := 50.0
		if maxS > minS {
			successScore = 100 * (g.SuccessRate - minS) / (maxS - minS)
		}
		speedScore := 50.0
		if g.SpeedMs > 0 {
			speedScore = 100000 / float64(g.SpeedMs)
			if speedScore > 100 {
				speedScore = 100
			}
		}
		wP, wS, wE := p.PriceWeight, p.SpeedWeight, p.SuccessWeight
		switch p.Strategy {
		case "price":
			wP, wS, wE = 100, 0, 0
		case "speed":
			wP, wS, wE = 15, 70, 15
		case "success":
			wP, wS, wE = 15, 15, 70
		}
		g.Score = priceScore*wP/100 + speedScore*wS/100 + successScore*wE/100
		g.Reason = fmt.Sprintf("score=%.2f price=%.4f success=%.2f", g.Score, g.Price, g.SuccessRate)
	}
	sortRankedGroups(out)
	return out, nil
}

// sortRankedGroups keeps the score as the primary decision signal and uses
// the persisted candidate order only for ties. This makes drag ordering
// deterministic without allowing a user preference to override hard metrics.
func sortRankedGroups(groups []Group) {
	sort.SliceStable(groups, func(i, j int) bool {
		if groups[i].Eligible != groups[j].Eligible {
			return groups[i].Eligible
		}
		if groups[i].Score != groups[j].Score {
			return groups[i].Score > groups[j].Score
		}
		if groups[i].SortOrder != groups[j].SortOrder {
			return groups[i].SortOrder < groups[j].SortOrder
		}
		return groups[i].GroupID < groups[j].GroupID
	})
}

func finiteNonNegative(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0
}

func (m *Module) explorationIndex(key *service.APIKey, p *Profile, ranked []Group, primaryGroupID int64, sessionHash, model string) int {
	if key == nil || p == nil || !finiteNonNegative(p.ExplorationRate) || p.ExplorationRate <= 0 || p.ExplorationRate > 0.05 || len(ranked) < 2 {
		return -1
	}
	digest := sha256.Sum256([]byte(fmt.Sprintf("%d:%d:%s:%s", key.ID, p.Version, sessionHash, model)))
	if float64(digest[0])/256 >= p.ExplorationRate {
		return -1
	}
	candidates := make([]int, 0, len(ranked))
	for i := range ranked {
		if !ranked[i].Eligible || ranked[i].GroupID == primaryGroupID || ranked[i].Samples >= minSamples {
			continue
		}
		candidates = append(candidates, i)
	}
	if len(candidates) == 0 {
		return -1
	}
	return candidates[int(digest[1])%len(candidates)]
}

// dispatchModelForGroup returns the model that the selected group's
// OpenAI-compatible Messages dispatcher will actually send upstream. The
// public Anthropic Messages endpoint must continue using the requested model;
// only OpenAI/Grok dispatch groups apply this compatibility mapping. Resolving
// it during candidate filtering prevents a valid OpenAI account that supports
// gpt-5.x from being rejected because it does not advertise the client-facing
// claude-* alias.
func (m *Module) dispatchModelForGroup(ctx context.Context, group *service.Group, model string) string {
	model = service.NormalizeOpenAICompatRequestedModel(strings.TrimSpace(model))
	if group == nil || model == "" {
		return model
	}
	state := service.SmartRoutingStateFromContext(ctx)
	if state == nil || state.EndpointType != "messages" {
		return model
	}
	if group.Platform != service.PlatformOpenAI && group.Platform != service.PlatformGrok {
		return model
	}
	if mapped := strings.TrimSpace(group.ResolveMessagesDispatchModel(model)); mapped != "" {
		return service.NormalizeOpenAICompatRequestedModel(mapped)
	}
	return model
}

func (m *Module) pricingComparableForGroups(ctx context.Context, primary, candidate *service.Group, model string) bool {
	if m == nil || primary == nil || candidate == nil {
		return false
	}
	if m.gateway == nil && m.openAI == nil {
		return true
	}
	check := func(groupID int64) bool {
		if m.gateway != nil {
			ok, err := m.gateway.CheckSmartRoutingPricingComparable(ctx, groupID, model)
			if err == nil && ok {
				return true
			}
		}
		if m.openAI != nil {
			ok, err := m.openAI.CheckSmartRoutingPricingComparable(ctx, groupID, model)
			if err == nil && ok {
				return true
			}
		}
		return false
	}
	if primary.ID == candidate.ID {
		return check(primary.ID) || m.explicitChannelPricingComparable(ctx, primary, candidate, model)
	}
	if check(primary.ID) && check(candidate.ID) {
		return true
	}
	return m.explicitChannelPricingComparable(ctx, primary, candidate, model)
}

// explicitChannelPricingComparable is the narrow escape from the legacy
// "any channel mapping means unknown price" rule. It only accepts a pair when
// both groups expose the same billing mode and every explicit unit price is
// known; the candidate must not be more expensive after its group multiplier.
func (m *Module) explicitChannelPricingComparable(ctx context.Context, primary, candidate *service.Group, model string) bool {
	lookup := func(groupID int64) *service.ChannelModelPricing {
		models := []string{model}
		if m.gateway != nil {
			gid := groupID
			if mapped, _ := m.gateway.ResolveChannelMappingAndRestrict(ctx, &gid, model); mapped.Mapped && strings.TrimSpace(mapped.MappedModel) != "" {
				models = append(models, strings.TrimSpace(mapped.MappedModel))
			}
		}
		if m.openAI != nil {
			gid := groupID
			if mapped, _ := m.openAI.ResolveChannelMappingAndRestrict(ctx, &gid, model); mapped.Mapped && strings.TrimSpace(mapped.MappedModel) != "" {
				models = append(models, strings.TrimSpace(mapped.MappedModel))
			}
		}
		for _, lookupModel := range models {
			if m.gateway != nil {
				if p := m.gateway.GetSmartRoutingChannelModelPricing(ctx, groupID, lookupModel); p != nil {
					return p
				}
			}
			if m.openAI != nil {
				if p := m.openAI.GetSmartRoutingChannelModelPricing(ctx, groupID, lookupModel); p != nil {
					return p
				}
			}
		}
		return nil
	}
	left, right := lookup(primary.ID), lookup(candidate.ID)
	if left == nil || right == nil || left.BillingMode != right.BillingMode {
		return false
	}
	leftMul, rightMul := primary.RateMultiplier, candidate.RateMultiplier
	if leftMul < 0 || rightMul < 0 {
		return false
	}
	compare := func(a, b *float64) bool {
		if a == nil || b == nil {
			return a == nil && b == nil
		}
		return *b*rightMul <= *a*leftMul+1e-12
	}
	return compare(left.InputPrice, right.InputPrice) && compare(left.OutputPrice, right.OutputPrice) && compare(left.PerRequestPrice, right.PerRequestPrice)
}

// longContextPricingComparable verifies that selecting a candidate cannot
// introduce a long-context surcharge that the primary route would not apply.
// Group.LongContextPricingEnabled is the normal gate. OpenAI accounts also
// have an explicit opt-in that can enable the same ladder even when the group
// toggle is off, so the schedulable account pool is part of this comparison.
// Unknown account state fails closed only when the candidate may enable the
// surcharge; a definitely-disabled candidate is safe even if the primary
// pool is temporarily unavailable.
func (m *Module) longContextPricingComparable(ctx context.Context, primary, candidate *service.Group) (bool, string) {
	primaryEnabled, primaryKnown := m.longContextPricingCapability(ctx, primary)
	candidateEnabled, candidateKnown := m.longContextPricingCapability(ctx, candidate)
	if !candidateKnown {
		return false, "long-context pricing unavailable for comparison"
	}
	if !candidateEnabled {
		return true, ""
	}
	if !primaryKnown {
		return false, "long-context pricing unavailable for comparison"
	}
	if !primaryEnabled {
		return false, "long-context pricing exceeds primary group"
	}
	return true, ""
}

// longContextPricingCapability returns whether a group can apply context-tier
// pricing for at least one schedulable account. The account flag is relevant
// only to OpenAI; other platforms are governed solely by the group toggle.
func (m *Module) longContextPricingCapability(ctx context.Context, group *service.Group) (enabled, known bool) {
	if group == nil {
		return false, false
	}
	if group.LongContextPricingEnabled || group.Platform != service.PlatformOpenAI {
		return group.LongContextPricingEnabled, true
	}
	if m == nil || m.accounts == nil {
		return false, false
	}
	accounts, err := m.accounts.ListSchedulableByGroupIDAndPlatform(ctx, group.ID, service.PlatformOpenAI)
	if err != nil {
		return false, false
	}
	if len(accounts) == 0 {
		// Account admission will reject an empty pool separately. Treat the
		// capability as known-disabled here so it cannot create a false price
		// mismatch while preserving the normal no-account reason.
		return false, true
	}
	seenEnabled, seenDisabled := false, false
	for i := range accounts {
		if accounts[i].IsOpenAILongContextBillingEnabled() {
			seenEnabled = true
		} else {
			seenDisabled = true
		}
	}
	if seenEnabled && seenDisabled {
		// A request may land on either account. There is no safe single price
		// baseline for routing, so force the candidate out of the pool.
		return false, false
	}
	return seenEnabled, true
}

// candidateEndpointAllowed mirrors endpoint-level group gates that run in the
// handlers. The primary group is checked before scheduling, while a smart
// candidate is selected later; applying the same hard gates here prevents a
// weaker candidate from bypassing those restrictions.
func candidateEndpointAllowed(ctx context.Context, group *service.Group) bool {
	if group == nil {
		return false
	}
	state := service.SmartRoutingStateFromContext(ctx)
	if state == nil {
		return true
	}
	if group.ClaudeCodeOnly && (state.EndpointType != "messages" || !service.IsClaudeCodeClient(ctx)) {
		return false
	}
	if state.EndpointType == "messages" && group.Platform == service.PlatformOpenAI && !group.AllowMessagesDispatch {
		return false
	}
	if state.ImageGenerationIntent && !service.GroupAllowsImageGeneration(group) {
		return false
	}
	return true
}

func normalizeRoutingEndpoint(endpoint string) string {
	switch endpoint {
	case "chat_completions", "responses", "messages":
		return endpoint
	case "/v1/chat/completions", "/chat/completions":
		return "chat_completions"
	case "/v1/responses", "/responses":
		return "responses"
	case "/v1/messages", "/messages":
		return "messages"
	default:
		return ""
	}
}
