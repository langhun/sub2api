package smartrouting

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
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
	APIKeyID      int64   `json:"api_key_id"`
	Enabled       bool    `json:"enabled"`
	Mode          string  `json:"mode"`
	Strategy      string  `json:"strategy"`
	PriceWeight   float64 `json:"price_weight"`
	SpeedWeight   float64 `json:"speed_weight"`
	SuccessWeight float64 `json:"success_weight"`
	// Deprecated: cross-group stickiness was removed; the field is retained
	// only so existing persisted profiles keep round-tripping.
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
	PriceScore    float64 `json:"price_score"`
	SpeedScore    float64 `json:"speed_score"`
	SuccessScore  float64 `json:"success_score"`
	Eligible      bool    `json:"eligible"`
	Reason        string  `json:"reason,omitempty"`
}

type UpdateRequest struct {
	Version       int64   `json:"version"`
	Enabled       bool    `json:"enabled"`
	Mode          string  `json:"mode"`
	Strategy      string  `json:"strategy"`
	PriceWeight   float64 `json:"price_weight"`
	SpeedWeight   float64 `json:"speed_weight"`
	SuccessWeight float64 `json:"success_weight"`
	// Deprecated: cross-group stickiness was removed; the field is retained
	// only so existing persisted profiles keep round-tripping.
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
	ctx, cancel := context.WithTimeout(ctx, 250*time.Millisecond)
	defer cancel()
	endpoint, stream := "", false
	if state := service.SmartRoutingStateFromContext(ctx); state != nil {
		endpoint, stream = state.EndpointType, state.Stream
	}
	// Keep the realtime and fallback windows in one query. A route with a cold
	// candidate can otherwise issue two DB round trips per group on every first
	// request, multiplying latency exactly when the pool is warming up.
	// A failed scan degrades every candidate to the cold-start baseline; log it
	// so stats outages don't silently turn routing into pure price ordering.
	if err := m.repo.db.QueryRowContext(ctx, `SELECT
		COUNT(*) FILTER (WHERE created_at>=NOW()-INTERVAL '15 minutes'),
		COUNT(*) FILTER (WHERE success AND created_at>=NOW()-INTERVAL '15 minutes'),
		COALESCE(percentile_cont(0.5) WITHIN GROUP (ORDER BY speed_ms) FILTER (WHERE success AND speed_ms>0 AND created_at>=NOW()-INTERVAL '15 minutes'),0)::bigint,
		COUNT(*), COUNT(*) FILTER (WHERE success),
		COALESCE(percentile_cont(0.5) WITHIN GROUP (ORDER BY speed_ms) FILTER (WHERE success AND speed_ms>0),0)::bigint
		FROM (
			SELECT DISTINCT ON (request_id, group_id) success, speed_ms, created_at
			FROM custom_route_attempts
			WHERE group_id=$1 AND requested_model=$2 AND endpoint_type=$3 AND stream=$4
			  AND created_at>=NOW()-INTERVAL '1 hour'
			ORDER BY request_id, group_id, attempt DESC
		) final_attempts`, groupID, model, endpoint, stream).Scan(&st.Samples, &st.Success, &st.SpeedMs, &st.FallbackSamples, &st.FallbackSuccess, &st.FallbackSpeedMs); err != nil {
		slog.Warn("smart_routing_stats_load_failed", "group_id", groupID, "model", model, "error", err)
	}
	return st
}

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

// ErrVersionConflict is returned when a save races another writer or omits
// the current version of an existing profile.
var ErrVersionConflict = errors.New("routing profile version conflict")

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
		return &Profile{APIKeyID: keyID, Mode: "single", Strategy: "auto", PriceWeight: 40, SpeedWeight: 30, SuccessWeight: 30, StickyTTL: 0, ExplorationRate: 0.05, Version: 0, Groups: []Group{}}, nil
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
	defer func() { _ = rows.Close() }()
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
	defer func() { _ = tx.Rollback() }()
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
			return nil, ErrVersionConflict
		}
	} else if !exists || current != req.Version {
		return nil, ErrVersionConflict
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

func routingWeights(p *Profile) (price, speed, success float64) {
	if p == nil {
		return service.SmartRoutingStrategyWeights("auto", 0, 0, 0)
	}
	return service.SmartRoutingStrategyWeights(p.Strategy, p.PriceWeight, p.SpeedWeight, p.SuccessWeight)
}

type Module struct {
	feedback routeFeedback
	repo     *Repository
	apiKeys  *service.APIKeyService
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
		// Only caller-fixable conflicts are client errors; DB failures and
		// post-commit reads must surface as 500, not a misleading 400.
		if errors.Is(err, ErrVersionConflict) {
			c.JSON(400, gin.H{"detail": err.Error()})
		} else {
			c.JSON(500, gin.H{"detail": err.Error()})
		}
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
	// An empty body is allowed (defaults apply), but malformed JSON must not be
	// silently treated as an empty request.
	if err := c.ShouldBindJSON(&req); err != nil && err != io.EOF {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid request: " + err.Error()})
		return
	}
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
		if state.ContinuityGroupID > 0 {
			state.NoRoute = true
		}
		defer func() {
			if !state.SmartRoutingActive && (state.Attempt > 0 || state.CapacityRetries > 0) {
				state.NoRoute = true
			}
		}()
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
	if state.ContinuityGroupID > 0 {
		state.NoRoute = false
	}
	state.Observer = m
	state.Strategy = p.Strategy
	state.PriceWeight, state.SpeedWeight, state.SuccessWeight = routingWeights(p)
	if state.NoRoute {
		return nil, nil
	}
	if mapped := service.OpenAIForwardModelFromContext(ctx); mapped != "" {
		model = mapped
	}
	state.RequestedModel = model
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
	// route, however. Mark an empty pool as NoRoute so restoring the primary
	// cannot accidentally re-enter scheduling for an exhausted request.
	hasEligible := false
	for _, candidate := range ranked {
		if candidate.Eligible {
			hasEligible = true
			break
		}
	}
	if !hasEligible {
		state.NoRoute = true
		m.recordNoRoute(ctx, key, p, model, ranked)
		return nil, nil
	}
	// Preserve account retries in the admitted group until it is explicitly
	// exhausted. A new request has no attempts and still gets a fresh ranking.
	if state.Attempt > 0 || state.CapacityRetries > 0 || len(excluded) > 0 {
		if _, exhausted := state.ExhaustedGroups[state.CurrentGroupID]; !exhausted {
			for i := range ranked {
				if ranked[i].GroupID == state.CurrentGroupID && ranked[i].Eligible {
					ranked[0], ranked[i] = ranked[i], ranked[0]
					break
				}
			}
		}
	}
	// Smart routing deliberately does not pin a request session to a group.
	// Re-score the primary and candidates on every request so a recovered,
	// lower-cost primary group is selected immediately. Upstream account-level
	// stickiness remains responsible for provider session continuity.
	requestID, _ := ctx.Value(ctxkey.RequestID).(string)
	if exploreIndex := m.explorationIndex(key, p, ranked, state.MainGroupID, requestID, model); requestID != "" && exploreIndex >= 0 && state.Attempt == 0 && state.CapacityRetries == 0 && len(state.ExhaustedGroups) == 0 {
		// Exploration is applied only after hard eligibility checks. It gives a
		// cold candidate a bounded opportunity without making routing
		// nondeterministic for a given request. Price is a soft signal, so
		// even price-only strategies may explore.
		ranked[0], ranked[exploreIndex] = ranked[exploreIndex], ranked[0]
	}
	for i := range ranked {
		g := &ranked[i]
		if !g.Eligible {
			continue
		}
		if _, exhausted := state.ExhaustedGroups[g.GroupID]; exhausted {
			g.Reason = "group exhausted in request"
			continue
		}
		if _, rejected := state.BillingRejectedGroups[g.GroupID]; rejected {
			g.Reason = "billing admission rejected"
			continue
		}
		if !routeStateAllowsGroup(state, g.GroupID, maxSwitches) {
			g.Reason = "continuity or switch budget restriction"
			// previous_response_id chains cannot be replayed against another
			// group. Let the normal selector return its explicit no-account
			// error instead of silently changing the conversation owner.
			continue
		}
		// Account failover is intentionally handled inside the current group
		// before moving to another group. The resolver marks a group as tried
		// when it is admitted, so this check must run before the TriedGroups
		// filter on subsequent scheduler calls.
		if g.GroupID == state.CurrentGroupID && (len(excluded) > 0 || state.Attempt > 0 || state.CapacityRetries > 0) {
			if m.accounts == nil {
				return nil, nil
			}
			// g.Platform is hydrated in rank(); key.Group is a mutable gateway
			// pointer that may be nil mid-failover, and an empty platform would
			// make the account lookup report zero remaining accounts.
			remaining, known := m.remainingModelAccountsKnown(ctx, state.CurrentGroupID, g.Platform, m.dispatchModelForGroup(ctx, key.Group, model), excluded)
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
			m.recordDecision(ctx, key, p, model, *g, decision)
			return decision, nil
		}
		if _, tried := state.TriedGroups[g.GroupID]; tried {
			g.Reason = "group already tried"
			continue
		}
		depth := len(state.TriedGroups)
		decision := &service.SmartRouteDecision{GroupID: g.GroupID, FallbackDepth: depth, Reason: g.Reason}
		m.recordDecision(ctx, key, p, model, *g, decision)
		return decision, nil
	}
	// Every eligible row may already be tried, rejected by continuity, or
	// beyond the switch budget. There is no next route for this invocation;
	// fail closed instead of reusing an exhausted primary group.
	state.NoRoute = true
	m.recordNoRoute(ctx, key, p, model, ranked)
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

func routeStateAllowsGroup(state *service.SmartRoutingRequestState, groupID int64, maxSwitches int) bool {
	if state == nil {
		return true
	}
	if state.ContinuityLocked {
		owner := state.ContinuityGroupID
		if owner == 0 {
			owner = state.MainGroupID
		}
		if groupID != owner {
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

// remainingModelAccountsKnown returns whether the repository response was
// authoritative. A repository failure must not be interpreted as "zero
// accounts" because that would route a request away from the authenticated
// primary group during an infrastructure incident. Only a successful query
// proving zero eligible accounts may make a group ineligible.
func (m *Module) remainingModelAccountsKnown(ctx context.Context, groupID int64, platform, model string, excluded map[int64]struct{}) (int, bool) {
	n, known, _ := m.inspectModelAccounts(ctx, groupID, platform, model, excluded)
	return n, known
}

func (m *Module) inspectModelAccounts(ctx context.Context, groupID int64, platform, model string, excluded map[int64]struct{}) (int, bool, float64) {
	if m == nil || m.accounts == nil {
		return 0, false, 0
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
		return 0, false, 0
	}
	if m.groups == nil {
		return 0, false, 0
	}
	group, groupErr := m.groups.GetByIDLite(ctx, groupID)
	if groupErr != nil || group == nil {
		return 0, false, 0
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
		return 0, true, 0
	}
	remaining := 0
	eligibleAccounts := make([]service.Account, 0, len(accounts))
	healthModel := model
	if state := service.SmartRoutingStateFromContext(ctx); state != nil && state.RequestedModel != "" {
		healthModel = state.RequestedModel
	}
	unavailable := m.unavailableRoutingAccounts(ctx, accounts, healthModel)
	for i := range accounts {
		account := &accounts[i]
		if unavailable[account.ID] {
			if state := service.SmartRoutingStateFromContext(ctx); state != nil {
				if state.FailedAccountIDs == nil {
					state.FailedAccountIDs = map[int64]struct{}{}
				}
				state.FailedAccountIDs[account.ID] = struct{}{}
			}
			continue
		}
		if state := service.SmartRoutingStateFromContext(ctx); state != nil {
			if _, failed := state.FailedAccountIDs[account.ID]; failed {
				continue
			}
		}
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
		eligibleAccounts = append(eligibleAccounts, *account)
	}
	capacity, known := 0.0, false
	if platform == service.PlatformOpenAI && m.openAI != nil {
		capacity, known = m.openAI.SmartRoutingCapacity(ctx, eligibleAccounts)
	} else if m.gateway != nil {
		capacity, known = m.gateway.SmartRoutingCapacity(ctx, eligibleAccounts)
	}
	if known && capacity <= 0 {
		return 0, true, 0
	}
	return remaining, true, capacity
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
		// During failover the gateway rewrites key.GroupID to the last candidate,
		// so state.MainGroupID is the only trustworthy baseline. Falling back to
		// key.GroupID here would score candidates against the wrong group.
		var err error
		main, err = m.groups.GetByIDLite(ctx, state.MainGroupID)
		if err != nil {
			return nil, fmt.Errorf("primary group %d unavailable: %w", state.MainGroupID, err)
		}
		if main == nil {
			return nil, fmt.Errorf("primary group %d not found", state.MainGroupID)
		}
	} else if main == nil && key.GroupID != nil {
		// First resolve on a request: key.GroupID is still the authed primary.
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
		remaining, known, capacity := m.inspectModelAccounts(ctx, main.ID, main.Platform, mainModel, excluded)
		mainCandidate.CapacityRatio = capacity
		if known && remaining == 0 {
			mainCandidate.Eligible = false
			mainCandidate.Reason = "no schedulable accounts"
		}
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
			// Candidate price is a scoring signal, not a hard eligibility gate.
			// A more expensive Plus group must still be available as a failover
			// when a discounted primary group is exhausted; actual billing uses
			// the group that serves the request.
			if m.accounts == nil {
				g.Reason = "account scheduler unavailable"
				out = append(out, g)
				continue
			}
			remaining, known, capacity := m.inspectModelAccounts(ctx, g.GroupID, main.Platform, candidateModel, excluded)
			g.CapacityRatio = capacity
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
		reliableSamples := st.Samples >= minSamples
		if !reliableSamples && st.FallbackSamples >= minSamples {
			st.Samples = st.FallbackSamples
			st.Success = st.FallbackSuccess
			st.SpeedMs = st.FallbackSpeedMs
			reliableSamples = true
		}
		g.Samples = st.Samples
		g.SuccessRate = 0.5
		if reliableSamples {
			g.SuccessRate = float64(st.Success+1) / float64(st.Samples+2)
		}
		if reliableSamples && st.SpeedMs.Valid {
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
	minSpeed, maxSpeed := int64(0), int64(0)
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
		if g.SpeedMs > 0 && (minSpeed == 0 || g.SpeedMs < minSpeed) {
			minSpeed = g.SpeedMs
		}
		if g.SpeedMs > maxSpeed {
			maxSpeed = g.SpeedMs
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
		speedScore := relativeSpeedScore(g.SpeedMs, minSpeed, maxSpeed)
		wP, wS, wE := routingWeights(p)
		g.PriceScore, g.SpeedScore, g.SuccessScore = priceScore, speedScore, successScore
		g.Score = priceScore*wP/100 + speedScore*wS/100 + successScore*wE/100
		g.Reason = fmt.Sprintf("score=%.2f price_score=%.2f speed_score=%.2f success_score=%.2f samples=%d", g.Score, priceScore, speedScore, successScore, g.Samples)
	}
	sortRankedGroups(out)
	return out, nil
}

func relativeSpeedScore(speedMs, minSpeed, maxSpeed int64) float64 {
	if speedMs <= 0 || maxSpeed <= 0 {
		return 50
	}
	if maxSpeed == minSpeed {
		return 100
	}
	if speedMs < minSpeed {
		speedMs = minSpeed
	}
	if speedMs > maxSpeed {
		speedMs = maxSpeed
	}
	return 100 * float64(maxSpeed-speedMs) / float64(maxSpeed-minSpeed)
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
		if groups[i].CapacityRatio != groups[j].CapacityRatio {
			return groups[i].CapacityRatio > groups[j].CapacityRatio
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

func (m *Module) explorationIndex(key *service.APIKey, p *Profile, ranked []Group, primaryGroupID int64, requestID, model string) int {
	if key == nil || p == nil || !finiteNonNegative(p.ExplorationRate) || p.ExplorationRate <= 0 || p.ExplorationRate > 0.05 || len(ranked) < 2 {
		return -1
	}
	digest := sha256.Sum256([]byte(fmt.Sprintf("%d:%d:%s:%s", key.ID, p.Version, requestID, model)))
	bucket := uint32(digest[0])<<16 | uint32(digest[1])<<8 | uint32(digest[2])
	if bucket >= uint32(p.ExplorationRate*(1<<24)) {
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
	return candidates[int(digest[3])%len(candidates)]
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
