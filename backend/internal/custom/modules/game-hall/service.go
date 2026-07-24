package gamehall

import (
	"context"
	cryptorand "crypto/rand"
	"fmt"
	"log/slog"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
)

const (
	GameTypeSlots        = "slots"
	GameSlotsRuleVersion = "slots-v1-rtp-95.3"

	GameExchangeBalanceToDG = "balance_to_dg"
	GameExchangeDGToBalance = "dg_to_balance"

	// Legacy setting keys remain a read-only compatibility contract until the
	// game-hall settings endpoint is migrated into this module.
	SettingKeyGameHallEnabled              = "game_hall_enabled"
	SettingKeyGameSlotsEnabled             = "game_slots_enabled"
	SettingKeyGameSlotsMinBet              = "game_slots_min_bet"
	SettingKeyGameSlotsMaxBet              = "game_slots_max_bet"
	SettingKeyGameExchangeMinAmount        = "game_exchange_min_amount"
	SettingKeyGameExchangeMaxAmount        = "game_exchange_max_amount"
	SettingKeyGameExchangeDailyLimit       = "game_exchange_daily_limit"
	SettingKeyGameExchangeAllowDGToBalance = "game_exchange_allow_dg_to_balance"
)

var (
	ErrGameHallDisabled             = infraerrors.Forbidden("GAME_HALL_DISABLED", "game hall is disabled")
	ErrGameHallUserDisabled         = infraerrors.Forbidden("GAME_HALL_USER_DISABLED", "game hall is disabled for this user")
	ErrGameExchangeAmountInvalid    = infraerrors.BadRequest("GAME_EXCHANGE_AMOUNT_INVALID", "exchange amount must be greater than 0")
	ErrGameExchangeDirectionInvalid = infraerrors.BadRequest("GAME_EXCHANGE_DIRECTION_INVALID", "exchange direction is invalid")
	ErrGameExchangeOutOfRange       = infraerrors.BadRequest("GAME_EXCHANGE_OUT_OF_RANGE", "exchange amount is outside the configured range")
	ErrGameExchangeDailyLimit       = infraerrors.BadRequest("GAME_EXCHANGE_DAILY_LIMIT", "daily exchange limit exceeded")
	ErrGameExchangeReturnDisabled   = infraerrors.Forbidden("GAME_EXCHANGE_RETURN_DISABLED", "DG to main balance exchange is disabled")
	ErrGameInsufficientMainBalance  = infraerrors.BadRequest("GAME_INSUFFICIENT_MAIN_BALANCE", "insufficient main balance")
	ErrGameInsufficientDGBalance    = infraerrors.BadRequest("GAME_INSUFFICIENT_DG_BALANCE", "insufficient DG balance")
	ErrGameInvalidType              = infraerrors.BadRequest("GAME_INVALID_TYPE", "game type is invalid")
	ErrGameInvalidBetAmount         = infraerrors.BadRequest("GAME_INVALID_BET_AMOUNT", "bet amount must be greater than 0")
	ErrGameSlotsDisabled            = infraerrors.Forbidden("GAME_SLOTS_DISABLED", "slots game is disabled")
	ErrGameBetOutOfRange            = infraerrors.BadRequest("GAME_BET_OUT_OF_RANGE", "bet amount is outside the configured range")
	ErrGameRandomUnavailable        = infraerrors.InternalServer("GAME_RANDOM_UNAVAILABLE", "secure random source is unavailable")

	slotRandomIntN = secureSlotIntN
)

type slotSymbolSpec struct {
	id      string
	weight  int
	payout3 float64
}

var slotSymbolTable = []slotSymbolSpec{
	{id: "cherry", weight: 25, payout3: 18.7},
	{id: "lemon", weight: 18, payout3: 30},
	{id: "orange", weight: 18, payout3: 30},
	{id: "grape", weight: 14, payout3: 48},
	{id: "bell", weight: 10, payout3: 72},
	{id: "star", weight: 7, payout3: 108},
	{id: "diamond", weight: 4, payout3: 180},
	{id: "seven", weight: 2, payout3: 320},
}

var slotTotalWeight = sumSlotWeights(slotSymbolTable)

const slotRuleVersion = "slots-v1"

type GameHallSettingsReader interface {
	GetMultiple(ctx context.Context, keys []string) (map[string]string, error)
}

type GameHallStore interface {
	GetSnapshot(ctx context.Context, userID int64) (*GameWalletSnapshot, error)
	CommitExchange(ctx context.Context, plan GameExchangePlan) (*GameExchangeResult, error)
	GetDailyExchangeTotal(ctx context.Context, userID int64, start, end time.Time) (float64, error)
	CommitSlotRound(ctx context.Context, plan GameSlotRoundPlan) (*GamePlayResult, error)
	ListWalletTransactions(ctx context.Context, userID *int64, page, pageSize int) ([]GameWalletTransaction, int64, error)
	ListRounds(ctx context.Context, userID *int64, page, pageSize int) ([]GameRound, int64, error)
}

type GameHallBalanceCache interface {
	InvalidateUserBalance(ctx context.Context, userID int64) error
}

type GameHallService struct {
	store        GameHallStore
	settings     GameHallSettingsReader
	balanceCache GameHallBalanceCache
	rollSlot     func() (float64, []string, string, error)
}

type GameWalletSnapshot struct {
	UserID           int64
	GameHallDisabled bool
	MainBalance      float64
	DGBalance        float64
	JackpotBalance   float64
}

type GameInfo struct {
	Type           string           `json:"type"`
	Name           string           `json:"name"`
	Description    string           `json:"description"`
	MinBet         float64          `json:"min_bet"`
	MaxBet         float64          `json:"max_bet"`
	Multipliers    []float64        `json:"multipliers"`
	RuleVersion    string           `json:"rule_version"`
	TheoreticalRTP float64          `json:"theoretical_rtp"`
	PayoutRules    []GamePayoutRule `json:"payout_rules"`
}

type GamePayoutRule struct {
	Symbol      string  `json:"symbol"`
	MatchCount  int     `json:"match_count"`
	Multiplier  float64 `json:"multiplier"`
	Probability float64 `json:"probability"`
}

type GameHallStatus struct {
	MainBalance              float64    `json:"main_balance"`
	DGBalance                float64    `json:"dg_balance"`
	JackpotBalance           float64    `json:"jackpot_balance"`
	Games                    []GameInfo `json:"games"`
	ExchangeMinAmount        float64    `json:"exchange_min_amount"`
	ExchangeMaxAmount        float64    `json:"exchange_max_amount"`
	ExchangeDailyLimit       float64    `json:"exchange_daily_limit"`
	ExchangeDailyUsed        float64    `json:"exchange_daily_used"`
	ExchangeDailyRemaining   float64    `json:"exchange_daily_remaining"`
	ExchangeAllowDGToBalance bool       `json:"exchange_allow_dg_to_balance"`
}

type GameExchangeInput struct {
	UserID         int64
	Direction      string
	Amount         float64
	IdempotencyKey string
}

type GameExchangePlan struct {
	UserID            int64
	Direction         string
	Amount            float64
	IdempotencyKey    string
	MainBalanceBefore float64
	MainBalanceAfter  float64
	DGBalanceBefore   float64
	DGBalanceAfter    float64
	DailyLimit        float64
	DayStart          time.Time
	DayEnd            time.Time
}

type GameExchangeResult struct {
	Direction         string  `json:"direction"`
	Amount            float64 `json:"amount"`
	MainBalanceBefore float64 `json:"main_balance_before"`
	MainBalanceAfter  float64 `json:"main_balance_after"`
	DGBalanceBefore   float64 `json:"dg_balance_before"`
	DGBalanceAfter    float64 `json:"dg_balance_after"`
}

type GamePlayInput struct {
	UserID         int64
	GameType       string
	BetAmount      float64
	IdempotencyKey string
}

type GameSlotRoundPlan struct {
	UserID          int64
	GameType        string
	BetAmount       float64
	PayoutAmount    float64
	NetAmount       float64
	Multiplier      float64
	DGBalanceBefore float64
	DGBalanceAfter  float64
	JackpotBefore   float64
	JackpotAfter    float64
	Symbols         []string
	Outcome         string
	Message         string
	IdempotencyKey  string
}

type GamePlayResult struct {
	RoundID         int64    `json:"round_id"`
	GameType        string   `json:"game_type"`
	BetAmount       float64  `json:"bet_amount"`
	PayoutAmount    float64  `json:"payout_amount"`
	NetAmount       float64  `json:"net_amount"`
	Multiplier      float64  `json:"multiplier"`
	DGBalanceBefore float64  `json:"dg_balance_before"`
	DGBalanceAfter  float64  `json:"dg_balance_after"`
	JackpotBalance  float64  `json:"jackpot_balance"`
	Outcome         string   `json:"outcome"`
	Symbols         []string `json:"symbols,omitempty"`
	Message         string   `json:"message"`
}

type GameWalletTransaction struct {
	ID            int64          `json:"id"`
	UserID        int64          `json:"user_id"`
	TxType        string         `json:"tx_type"`
	Amount        float64        `json:"amount"`
	BalanceBefore float64        `json:"balance_before"`
	BalanceAfter  float64        `json:"balance_after"`
	ReferenceType string         `json:"reference_type"`
	ReferenceID   string         `json:"reference_id"`
	Metadata      map[string]any `json:"metadata"`
	CreatedAt     time.Time      `json:"created_at"`
}

type GameRound struct {
	ID            int64          `json:"id"`
	UserID        int64          `json:"user_id"`
	GameType      string         `json:"game_type"`
	BetAmount     float64        `json:"bet_amount"`
	PayoutAmount  float64        `json:"payout_amount"`
	NetAmount     float64        `json:"net_amount"`
	Multiplier    float64        `json:"multiplier"`
	BalanceBefore float64        `json:"balance_before"`
	BalanceAfter  float64        `json:"balance_after"`
	JackpotBefore float64        `json:"jackpot_before"`
	JackpotAfter  float64        `json:"jackpot_after"`
	Outcome       string         `json:"outcome"`
	Symbols       []string       `json:"symbols"`
	Metadata      map[string]any `json:"metadata"`
	CreatedAt     time.Time      `json:"created_at"`
}

func NewGameHallService(store GameHallStore, settings GameHallSettingsReader, balanceCache ...GameHallBalanceCache) *GameHallService {
	var cache GameHallBalanceCache
	if len(balanceCache) > 0 {
		cache = balanceCache[0]
	}
	return &GameHallService{
		store:        store,
		settings:     settings,
		balanceCache: cache,
		rollSlot:     defaultSlotRoller,
	}
}

func (s *GameHallService) SetSlotRoller(roller func() (float64, []string, string, error)) {
	if roller != nil {
		s.rollSlot = roller
	}
}

func (s *GameHallService) GetHallStatus(ctx context.Context, userID int64) (*GameHallStatus, error) {
	settings, err := s.readSettings(ctx)
	if err != nil {
		return nil, err
	}

	snapshot, err := s.store.GetSnapshot(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get game hall snapshot: %w", err)
	}
	if snapshot.GameHallDisabled {
		return nil, ErrGameHallUserDisabled
	}
	dailyTotal, err := s.dailyExchangeTotal(ctx, userID)
	if err != nil {
		return nil, err
	}
	dailyRemaining := float64(0)
	if settings.exchangeDailyLimit > 0 {
		dailyRemaining = roundGameAmount(math.Max(0, settings.exchangeDailyLimit-dailyTotal))
	}

	return &GameHallStatus{
		MainBalance:              roundGameAmount(snapshot.MainBalance),
		DGBalance:                roundGameAmount(snapshot.DGBalance),
		JackpotBalance:           roundGameAmount(snapshot.JackpotBalance),
		Games:                    configuredGameInfos(settings),
		ExchangeMinAmount:        settings.exchangeMinAmount,
		ExchangeMaxAmount:        settings.exchangeMaxAmount,
		ExchangeDailyLimit:       settings.exchangeDailyLimit,
		ExchangeDailyUsed:        dailyTotal,
		ExchangeDailyRemaining:   dailyRemaining,
		ExchangeAllowDGToBalance: settings.exchangeAllowDGToBalance,
	}, nil
}

func (s *GameHallService) Exchange(ctx context.Context, input GameExchangeInput) (*GameExchangeResult, error) {
	settings, err := s.readSettings(ctx)
	if err != nil {
		return nil, err
	}

	amount := roundGameAmount(input.Amount)
	if amount <= 0 || math.IsNaN(amount) || math.IsInf(amount, 0) {
		return nil, ErrGameExchangeAmountInvalid
	}
	if amount < settings.exchangeMinAmount || (settings.exchangeMaxAmount > 0 && amount > settings.exchangeMaxAmount) {
		return nil, ErrGameExchangeOutOfRange
	}
	if input.Direction == GameExchangeDGToBalance && !settings.exchangeAllowDGToBalance {
		return nil, ErrGameExchangeReturnDisabled
	}
	key, err := normalizeGameHallIdempotencyKey(input.IdempotencyKey)
	if err != nil {
		return nil, err
	}

	snapshot, err := s.store.GetSnapshot(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("get game hall snapshot: %w", err)
	}
	if snapshot.GameHallDisabled {
		return nil, ErrGameHallUserDisabled
	}

	plan := GameExchangePlan{
		UserID:            input.UserID,
		Direction:         input.Direction,
		Amount:            amount,
		IdempotencyKey:    key,
		MainBalanceBefore: roundGameAmount(snapshot.MainBalance),
		DGBalanceBefore:   roundGameAmount(snapshot.DGBalance),
		DailyLimit:        settings.exchangeDailyLimit,
	}
	plan.DayStart = timezone.Today()
	plan.DayEnd = plan.DayStart.AddDate(0, 0, 1)

	switch input.Direction {
	case GameExchangeBalanceToDG:
		if snapshot.MainBalance < amount {
			return nil, ErrGameInsufficientMainBalance
		}
		plan.MainBalanceAfter = roundGameAmount(snapshot.MainBalance - amount)
		plan.DGBalanceAfter = roundGameAmount(snapshot.DGBalance + amount)
	case GameExchangeDGToBalance:
		if snapshot.DGBalance < amount {
			return nil, ErrGameInsufficientDGBalance
		}
		plan.MainBalanceAfter = roundGameAmount(snapshot.MainBalance + amount)
		plan.DGBalanceAfter = roundGameAmount(snapshot.DGBalance - amount)
	default:
		return nil, ErrGameExchangeDirectionInvalid
	}

	result, err := s.store.CommitExchange(ctx, plan)
	if err != nil {
		return nil, err
	}
	if s.balanceCache != nil {
		if err := s.balanceCache.InvalidateUserBalance(context.WithoutCancel(ctx), input.UserID); err != nil {
			slog.Warn("game hall exchange committed but balance cache invalidation failed", "user_id", input.UserID, "error", err)
		}
	}
	return result, nil
}

func (s *GameHallService) ListUserTransactions(ctx context.Context, userID int64, page, pageSize int) ([]GameWalletTransaction, int64, error) {
	if err := s.ensureEnabled(ctx); err != nil {
		return nil, 0, err
	}
	return s.store.ListWalletTransactions(ctx, &userID, normalizeGamePage(page), normalizeGamePageSize(pageSize))
}

func (s *GameHallService) ListUserRounds(ctx context.Context, userID int64, page, pageSize int) ([]GameRound, int64, error) {
	if err := s.ensureEnabled(ctx); err != nil {
		return nil, 0, err
	}
	return s.store.ListRounds(ctx, &userID, normalizeGamePage(page), normalizeGamePageSize(pageSize))
}

func (s *GameHallService) ListAdminTransactions(ctx context.Context, userID *int64, page, pageSize int) ([]GameWalletTransaction, int64, error) {
	return s.store.ListWalletTransactions(ctx, userID, normalizeGamePage(page), normalizeGamePageSize(pageSize))
}

func (s *GameHallService) ListAdminRounds(ctx context.Context, userID *int64, page, pageSize int) ([]GameRound, int64, error) {
	return s.store.ListRounds(ctx, userID, normalizeGamePage(page), normalizeGamePageSize(pageSize))
}

func normalizeGamePage(page int) int {
	if page < 1 {
		return 1
	}
	return page
}

func normalizeGamePageSize(pageSize int) int {
	if pageSize < 1 || pageSize > 100 {
		return 20
	}
	return pageSize
}

func (s *GameHallService) Play(ctx context.Context, input GamePlayInput) (*GamePlayResult, error) {
	settings, err := s.readSettings(ctx)
	if err != nil {
		return nil, err
	}
	if input.GameType != GameTypeSlots {
		return nil, ErrGameInvalidType
	}
	if !settings.slotsEnabled {
		return nil, ErrGameSlotsDisabled
	}

	betAmount := roundGameAmount(input.BetAmount)
	if betAmount <= 0 || math.IsNaN(betAmount) || math.IsInf(betAmount, 0) {
		return nil, ErrGameInvalidBetAmount
	}
	if betAmount < settings.minBet || betAmount > settings.maxBet {
		return nil, ErrGameBetOutOfRange
	}
	key, err := normalizeGameHallIdempotencyKey(input.IdempotencyKey)
	if err != nil {
		return nil, err
	}

	snapshot, err := s.store.GetSnapshot(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("get game hall snapshot: %w", err)
	}
	if snapshot.GameHallDisabled {
		return nil, ErrGameHallUserDisabled
	}
	if snapshot.DGBalance < betAmount {
		return nil, ErrGameInsufficientDGBalance
	}

	multiplier, symbols, message, err := s.rollSlot()
	if err != nil {
		return nil, ErrGameRandomUnavailable.WithCause(err)
	}
	payoutAmount := roundGameAmount(betAmount * multiplier)
	netAmount := roundGameAmount(payoutAmount - betAmount)
	dgBalanceAfter := roundGameAmount(snapshot.DGBalance - betAmount + payoutAmount)
	jackpotAfter := roundGameAmount(snapshot.JackpotBalance + betAmount - payoutAmount)
	if jackpotAfter < 0 {
		jackpotAfter = 0
	}

	plan := GameSlotRoundPlan{
		UserID:          input.UserID,
		GameType:        input.GameType,
		BetAmount:       betAmount,
		PayoutAmount:    payoutAmount,
		NetAmount:       netAmount,
		Multiplier:      multiplier,
		DGBalanceBefore: roundGameAmount(snapshot.DGBalance),
		DGBalanceAfter:  dgBalanceAfter,
		JackpotBefore:   roundGameAmount(snapshot.JackpotBalance),
		JackpotAfter:    jackpotAfter,
		Symbols:         symbols,
		Outcome:         resolveGameOutcome(netAmount),
		Message:         message,
		IdempotencyKey:  key,
	}

	return s.store.CommitSlotRound(ctx, plan)
}

func (s *GameHallService) ensureEnabled(ctx context.Context) error {
	_, err := s.readSettings(ctx)
	return err
}

type gameHallRuntimeSettings struct {
	slotsEnabled             bool
	minBet                   float64
	maxBet                   float64
	exchangeMinAmount        float64
	exchangeMaxAmount        float64
	exchangeDailyLimit       float64
	exchangeAllowDGToBalance bool
}

func (s *GameHallService) readSettings(ctx context.Context) (gameHallRuntimeSettings, error) {
	if s == nil || s.settings == nil {
		return gameHallRuntimeSettings{}, ErrGameHallDisabled
	}
	if s.store == nil {
		return gameHallRuntimeSettings{}, ErrGameHallDisabled
	}
	values, err := s.settings.GetMultiple(ctx, []string{
		SettingKeyGameHallEnabled, SettingKeyGameSlotsEnabled, SettingKeyGameSlotsMinBet, SettingKeyGameSlotsMaxBet,
		SettingKeyGameExchangeMinAmount, SettingKeyGameExchangeMaxAmount, SettingKeyGameExchangeDailyLimit, SettingKeyGameExchangeAllowDGToBalance,
	})
	if err != nil {
		return gameHallRuntimeSettings{}, err
	}
	if values[SettingKeyGameHallEnabled] != "true" {
		return gameHallRuntimeSettings{}, ErrGameHallDisabled
	}
	minBet := parseBalanceFeatureFloat(values[SettingKeyGameSlotsMinBet], 0.01)
	maxBet := parseBalanceFeatureFloat(values[SettingKeyGameSlotsMaxBet], 1000)
	if minBet <= 0 || maxBet < minBet {
		return gameHallRuntimeSettings{}, infraerrors.InternalServer("GAME_SETTINGS_INVALID", "game hall bet range is invalid")
	}
	exchangeMinAmount := parseBalanceFeatureFloat(values[SettingKeyGameExchangeMinAmount], 0.01)
	exchangeMaxAmount := parseBalanceFeatureFloat(values[SettingKeyGameExchangeMaxAmount], 1000)
	exchangeDailyLimit := parseBalanceFeatureFloat(values[SettingKeyGameExchangeDailyLimit], 1000)
	if exchangeMinAmount <= 0 || (exchangeMaxAmount > 0 && exchangeMaxAmount < exchangeMinAmount) {
		return gameHallRuntimeSettings{}, infraerrors.InternalServer("GAME_SETTINGS_INVALID", "game hall exchange range is invalid")
	}
	return gameHallRuntimeSettings{
		slotsEnabled: values[SettingKeyGameSlotsEnabled] == "true", minBet: minBet, maxBet: maxBet,
		exchangeMinAmount: exchangeMinAmount, exchangeMaxAmount: exchangeMaxAmount, exchangeDailyLimit: exchangeDailyLimit,
		exchangeAllowDGToBalance: values[SettingKeyGameExchangeAllowDGToBalance] != "false",
	}, nil
}

func (s *GameHallService) dailyExchangeTotal(ctx context.Context, userID int64) (float64, error) {
	start := timezone.Today()
	total, err := s.store.GetDailyExchangeTotal(ctx, userID, start, start.AddDate(0, 0, 1))
	if err != nil {
		return 0, fmt.Errorf("get daily game exchange total: %w", err)
	}
	return roundGameAmount(total), nil
}

func configuredGameInfos(settings gameHallRuntimeSettings) []GameInfo {
	if !settings.slotsEnabled {
		return []GameInfo{}
	}
	return []GameInfo{
		{
			Type:           GameTypeSlots,
			Name:           "Slots",
			Description:    "Three reels with instant DG settlement.",
			MinBet:         settings.minBet,
			MaxBet:         settings.maxBet,
			Multipliers:    []float64{0, 18.7, 30, 48, 72, 108, 180, 320},
			RuleVersion:    slotRuleVersion,
			TheoreticalRTP: roundGameAmount(slotTheoreticalRTP()),
			PayoutRules:    configuredSlotPayoutRules(),
		},
	}
}

func configuredSlotPayoutRules() []GamePayoutRule {
	rules := make([]GamePayoutRule, 0, len(slotSymbolTable))
	for _, symbol := range slotSymbolTable {
		singleProbability := float64(symbol.weight) / float64(slotTotalWeight)
		rules = append(rules, GamePayoutRule{
			Symbol:      symbol.id,
			MatchCount:  3,
			Multiplier:  symbol.payout3,
			Probability: roundGameAmount(singleProbability * singleProbability * singleProbability),
		})
	}
	return rules
}

func slotTheoreticalRTP() float64 {
	rtp := 0.0
	for _, symbol := range slotSymbolTable {
		probability := float64(symbol.weight) / float64(slotTotalWeight)
		rtp += probability * probability * probability * symbol.payout3
	}
	return rtp
}

func roundGameAmount(value float64) float64 {
	return math.Round(value*1e8) / 1e8
}

func normalizeGameHallIdempotencyKey(raw string) (string, error) {
	key, err := service.NormalizeIdempotencyKey(raw)
	if err != nil {
		return "", err
	}
	// The HTTP coordinator normally requires a caller-supplied key. Keep the
	// repository safe for direct service calls and coordinator-disabled startup:
	// an empty key must not collapse every operation for a user into one replay.
	if key == "" {
		return uuid.NewString(), nil
	}
	return key, nil
}

func parseBalanceFeatureFloat(raw string, fallback float64) float64 {
	value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil || value < 0 || math.IsNaN(value) || math.IsInf(value, 0) {
		return fallback
	}
	return value
}

func resolveGameOutcome(netAmount float64) string {
	switch {
	case netAmount > 0:
		return "win"
	case netAmount < 0:
		return "lose"
	default:
		return "draw"
	}
}

func defaultSlotRoller() (float64, []string, string, error) {
	return rollSlotWithIntN(slotRandomIntN)
}

func rollSlotWithIntN(intN func(int) (int, error)) (float64, []string, string, error) {
	symbols := make([]string, 3)
	selected := make([]slotSymbolSpec, 3)

	for index := range 3 {
		symbol, err := pickWeightedSlotSymbol(intN)
		if err != nil {
			return 0, nil, "", err
		}
		selected[index] = symbol
		symbols[index] = symbol.id
	}

	if selected[0].id == selected[1].id && selected[1].id == selected[2].id {
		return selected[0].payout3, symbols, "中奖", nil
	}

	return 0, symbols, "未中奖", nil
}

func pickWeightedSlotSymbol(intN func(int) (int, error)) (slotSymbolSpec, error) {
	roll, err := intN(slotTotalWeight)
	if err != nil {
		return slotSymbolSpec{}, err
	}
	cumulative := 0

	for _, symbol := range slotSymbolTable {
		cumulative += symbol.weight
		if roll < cumulative {
			return symbol, nil
		}
	}

	return slotSymbolTable[len(slotSymbolTable)-1], nil
}

func secureSlotIntN(max int) (int, error) {
	if max <= 0 {
		return 0, fmt.Errorf("random upper bound must be positive")
	}
	value, err := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0, fmt.Errorf("read secure random source: %w", err)
	}
	return int(value.Int64()), nil
}

func sumSlotWeights(symbols []slotSymbolSpec) int {
	total := 0
	for _, symbol := range symbols {
		total += symbol.weight
	}
	return total
}
