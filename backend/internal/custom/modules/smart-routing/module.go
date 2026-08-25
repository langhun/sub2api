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
	Samples int64
	Success int64
	SpeedMs sql.NullInt64
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
		return &Profile{APIKeyID: keyID, Mode: "single", Strategy: "auto", PriceWeight: 40, SpeedWeight: 30, SuccessWeight: 30, StickyTTL: 1800, ExplorationRate: 0.05, Version: 1}, nil
	}
	if err != nil {
		return nil, err
	}
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
	if err := validateUpdate(req); err != nil {
		return nil, err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if req.Version > 0 {
		var current int64
		err = tx.QueryRowContext(ctx, `SELECT version FROM custom_api_key_route_profiles WHERE api_key_id=$1 FOR UPDATE`, keyID).Scan(&current)
		if err == nil && current != req.Version {
			return nil, fmt.Errorf("routing profile version conflict")
		}
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
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
	if req.Mode != "single" && req.Mode != "smart" {
		return fmt.Errorf("mode must be single or smart")
	}
	if req.Strategy != "auto" && req.Strategy != "price" && req.Strategy != "speed" && req.Strategy != "success" && req.Strategy != "custom" {
		return fmt.Errorf("invalid strategy")
	}
	if req.PriceWeight < 0 || req.SpeedWeight < 0 || req.SuccessWeight < 0 || math.Abs(req.PriceWeight+req.SpeedWeight+req.SuccessWeight-100) > 0.001 {
		return fmt.Errorf("weights must be non-negative and sum to 100")
	}
	if req.StickyTTL < 0 || req.StickyTTL > 86400 {
		return fmt.Errorf("sticky_ttl_seconds out of range")
	}
	if req.ExplorationRate < 0 || req.ExplorationRate > 0.05 {
		return fmt.Errorf("exploration_rate must be between 0 and 0.05")
	}
	if req.Mode == "smart" && len(req.GroupIDs) == 0 {
		return fmt.Errorf("smart mode requires at least one group")
	}
	if len(req.GroupIDs) > defaultMaxCandidates {
		return fmt.Errorf("too many candidate groups")
	}
	return nil
}

type Module struct {
	repo     *Repository
	apiKeys  *service.APIKeyService
	groups   service.GroupRepository
	accounts service.AccountRepository
	gateway  *service.GatewayService
	openAI   *service.OpenAIGatewayService
	redis    *redis.Client
}

func NewModule(db *sql.DB, apiKeys *service.APIKeyService, groups service.GroupRepository, accounts service.AccountRepository, gateway *service.GatewayService, openAI *service.OpenAIGatewayService, redisClient *redis.Client) *Module {
	m := &Module{repo: NewRepository(db), apiKeys: apiKeys, groups: groups, accounts: accounts, gateway: gateway, openAI: openAI, redis: redisClient}
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
	allowed, err := m.apiKeys.GetAvailableGroups(ctx.Request.Context(), key.UserID)
	if err != nil {
		return err
	}
	allowedMap := map[int64]service.Group{}
	for _, g := range allowed {
		allowedMap[g.ID] = g
	}
	main := key.Group
	for _, gid := range req.GroupIDs {
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
	p, err := m.repo.Get(c.Request.Context(), key.ID)
	if err != nil {
		c.JSON(500, gin.H{"detail": err.Error()})
		return
	}
	ranked, err := m.rank(c.Request.Context(), key, p, req.Model, map[int64]struct{}{})
	if err != nil {
		c.JSON(400, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(200, gin.H{"profile": p, "groups": ranked})
}

func (m *Module) Resolve(ctx context.Context, key *service.APIKey, model, sessionHash string, excluded map[int64]struct{}) (*service.SmartRouteDecision, error) {
	if key == nil || key.GroupID == nil {
		return nil, nil
	}
	p, err := m.repo.Get(ctx, key.ID)
	if err != nil || p == nil || !p.Enabled || p.Mode != "smart" {
		return nil, err
	}
	state := service.SmartRoutingStateFromContext(ctx)
	if state == nil {
		return nil, nil
	}
	if state.EndpointType != "chat_completions" && state.EndpointType != "responses" && state.EndpointType != "messages" {
		return nil, nil
	}
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
		return nil, err
	}
	stickyID := m.getStickyGroup(ctx, key.ID, p.Version, sessionHash)
	if stickyID > 0 {
		for i := range ranked {
			if ranked[i].GroupID == stickyID && ranked[i].Eligible {
				ranked[0], ranked[i] = ranked[i], ranked[0]
				break
			}
		}
	}
	for _, g := range ranked {
		if !g.Eligible {
			continue
		}
		if _, tried := state.TriedGroups[g.GroupID]; tried {
			continue
		}
		if !routeStateAllowsGroup(state, g.GroupID, stickyID, maxSwitches) {
			// previous_response_id chains cannot be replayed against another
			// group. Let the normal selector return its explicit no-account
			// error instead of silently changing the conversation owner.
			continue
		}
		if g.GroupID == state.CurrentGroupID && len(excluded) > 0 {
			platform := ""
			if key.Group != nil {
				platform = key.Group.Platform
			}
			if m.accounts == nil {
				return nil, nil
			}
			remaining := m.remainingModelAccounts(ctx, state.CurrentGroupID, platform, model, excluded)
			if remaining <= 0 {
				state.TriedGroups[g.GroupID] = struct{}{}
				continue
			}
		}
		if state.CurrentGroupID > 0 && len(excluded) > 0 {
			platform := ""
			if key.Group != nil {
				platform = key.Group.Platform
			}
			if m.remainingModelAccounts(ctx, state.CurrentGroupID, platform, model, excluded) > 0 {
				return nil, nil
			}
		}
		depth := len(state.TriedGroups)
		m.setStickyGroup(ctx, key.ID, p.Version, sessionHash, g.GroupID, time.Duration(p.StickyTTL)*time.Second)
		decision := &service.SmartRouteDecision{GroupID: g.GroupID, FallbackDepth: depth, Reason: g.Reason}
		m.recordDecision(ctx, key, p, model, g, decision)
		return decision, nil
	}
	return nil, nil
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
	return state.CurrentGroupID <= 0 || groupID == state.CurrentGroupID || state.GroupSwitches < maxSwitches
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
	if m.accounts == nil {
		return 0
	}
	accounts, err := m.accounts.ListSchedulableByGroupIDAndPlatform(ctx, groupID, platform)
	if err != nil {
		return 0
	}
	remaining := 0
	for i := range accounts {
		account := &accounts[i]
		if _, skip := excluded[account.ID]; skip {
			continue
		}
		if model != "" && (!account.IsModelSupported(model) || !account.IsSchedulableForModelWithContext(ctx, model)) {
			continue
		}
		remaining++
	}
	return remaining
}

func (m *Module) rank(ctx context.Context, key *service.APIKey, p *Profile, model string, excluded map[int64]struct{}) ([]Group, error) {
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
	if m.apiKeys != nil {
		available, err := m.apiKeys.GetAvailableGroups(ctx, key.UserID)
		if err != nil {
			return nil, fmt.Errorf("refresh routing group permissions: %w", err)
		}
		for _, group := range available {
			allowed[group.ID] = true
		}
	}
	now := time.Now()
	mainEffectivePrice := main.RateMultiplier * main.PeakMultiplierAt(now)
	candidates := append([]Group{{GroupID: main.ID, Name: main.Name, Platform: main.Platform, Price: mainEffectivePrice, Eligible: true}}, p.Groups...)
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
			g.Name, g.Platform, g.Price = full.Name, full.Platform, full.RateMultiplier*full.PeakMultiplierAt(now)
			if full.Status != service.StatusActive || full.Platform != main.Platform || full.SubscriptionType != main.SubscriptionType || full.Platform == service.PlatformComposite {
				g.Reason = "incompatible group"
				out = append(out, g)
				continue
			}
			if m.apiKeys != nil && !allowed[g.GroupID] {
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
			if m.remainingModelAccounts(ctx, g.GroupID, main.Platform, model, excluded) == 0 {
				g.Reason = "no schedulable accounts"
				out = append(out, g)
				continue
			}
		}
		st := stats{}
		if m.repo.db != nil {
			_ = m.repo.db.QueryRowContext(ctx, `SELECT COUNT(*) FILTER (WHERE ul.duration_ms IS NOT NULL), COUNT(*) FILTER (WHERE ul.duration_ms IS NOT NULL AND NOT EXISTS (SELECT 1 FROM ops_error_logs oe WHERE oe.request_id=ul.request_id)), COALESCE(percentile_cont(0.5) WITHIN GROUP (ORDER BY ul.first_token_ms) FILTER (WHERE ul.first_token_ms IS NOT NULL),0)::bigint FROM usage_logs ul WHERE ul.group_id=$1 AND ($2='' OR ul.requested_model=$2) AND ul.created_at>=NOW()-INTERVAL '15 minutes'`, g.GroupID, model).Scan(&st.Samples, &st.Success, &st.SpeedMs)
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
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Eligible != out[j].Eligible {
			return out[i].Eligible
		}
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return out[i].GroupID < out[j].GroupID
	})
	return out, nil
}
