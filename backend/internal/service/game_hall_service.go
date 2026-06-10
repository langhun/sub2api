package service

import (
	"context"
	"fmt"
	"math"
	"math/rand/v2"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	GameTypeSlots = "slots"

	GameExchangeBalanceToDG = "balance_to_dg"
	GameExchangeDGToBalance = "dg_to_balance"
)

var (
	ErrGameHallDisabled             = infraerrors.Forbidden("GAME_HALL_DISABLED", "game hall is disabled")
	ErrGameExchangeAmountInvalid    = infraerrors.BadRequest("GAME_EXCHANGE_AMOUNT_INVALID", "exchange amount must be greater than 0")
	ErrGameExchangeDirectionInvalid = infraerrors.BadRequest("GAME_EXCHANGE_DIRECTION_INVALID", "exchange direction is invalid")
	ErrGameInsufficientMainBalance  = infraerrors.BadRequest("GAME_INSUFFICIENT_MAIN_BALANCE", "insufficient main balance")
	ErrGameInsufficientDGBalance    = infraerrors.BadRequest("GAME_INSUFFICIENT_DG_BALANCE", "insufficient DG balance")
	ErrGameInvalidType              = infraerrors.BadRequest("GAME_INVALID_TYPE", "game type is invalid")
	ErrGameInvalidBetAmount         = infraerrors.BadRequest("GAME_INVALID_BET_AMOUNT", "bet amount must be greater than 0")

	slotRandomIntN = rand.IntN
)

type slotSymbolSpec struct {
	id      string
	weight  int
	payout3 float64
}

var slotSymbolTable = []slotSymbolSpec{
	{id: "cherry", weight: 25, payout3: 3},
	{id: "lemon", weight: 18, payout3: 5},
	{id: "orange", weight: 18, payout3: 5},
	{id: "grape", weight: 14, payout3: 8},
	{id: "bell", weight: 10, payout3: 12},
	{id: "star", weight: 7, payout3: 18},
	{id: "diamond", weight: 4, payout3: 30},
	{id: "seven", weight: 2, payout3: 50},
}

var slotTotalWeight = sumSlotWeights(slotSymbolTable)

type GameHallSettingsReader interface {
	GetMultiple(ctx context.Context, keys []string) (map[string]string, error)
}

type GameHallStore interface {
	GetSnapshot(ctx context.Context, userID int64) (*GameWalletSnapshot, error)
	CommitExchange(ctx context.Context, plan GameExchangePlan) (*GameExchangeResult, error)
	CommitSlotRound(ctx context.Context, plan GameSlotRoundPlan) (*GamePlayResult, error)
}

type GameHallService struct {
	store    GameHallStore
	settings GameHallSettingsReader
	rollSlot func() (float64, []string, string)
}

type GameWalletSnapshot struct {
	UserID         int64
	MainBalance    float64
	DGBalance      float64
	JackpotBalance float64
}

type GameInfo struct {
	Type        string    `json:"type"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	MinBet      float64   `json:"min_bet"`
	MaxBet      float64   `json:"max_bet"`
	Multipliers []float64 `json:"multipliers"`
}

type GameHallStatus struct {
	MainBalance    float64    `json:"main_balance"`
	DGBalance      float64    `json:"dg_balance"`
	JackpotBalance float64    `json:"jackpot_balance"`
	Games          []GameInfo `json:"games"`
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

func NewGameHallService(store GameHallStore, settings GameHallSettingsReader) *GameHallService {
	return &GameHallService{
		store:    store,
		settings: settings,
		rollSlot: defaultSlotRoller,
	}
}

func (s *GameHallService) SetSlotRoller(roller func() (float64, []string, string)) {
	if roller != nil {
		s.rollSlot = roller
	}
}

func (s *GameHallService) GetHallStatus(ctx context.Context, userID int64) (*GameHallStatus, error) {
	if err := s.ensureEnabled(ctx); err != nil {
		return nil, err
	}

	snapshot, err := s.store.GetSnapshot(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get game hall snapshot: %w", err)
	}

	return &GameHallStatus{
		MainBalance:    roundGameAmount(snapshot.MainBalance),
		DGBalance:      roundGameAmount(snapshot.DGBalance),
		JackpotBalance: roundGameAmount(snapshot.JackpotBalance),
		Games:          defaultGameInfos(),
	}, nil
}

func (s *GameHallService) Exchange(ctx context.Context, input GameExchangeInput) (*GameExchangeResult, error) {
	if err := s.ensureEnabled(ctx); err != nil {
		return nil, err
	}

	amount := roundGameAmount(input.Amount)
	if amount <= 0 || math.IsNaN(amount) || math.IsInf(amount, 0) {
		return nil, ErrGameExchangeAmountInvalid
	}
	key, err := NormalizeIdempotencyKey(input.IdempotencyKey)
	if err != nil {
		return nil, err
	}

	snapshot, err := s.store.GetSnapshot(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("get game hall snapshot: %w", err)
	}

	plan := GameExchangePlan{
		UserID:            input.UserID,
		Direction:         input.Direction,
		Amount:            amount,
		IdempotencyKey:    key,
		MainBalanceBefore: roundGameAmount(snapshot.MainBalance),
		DGBalanceBefore:   roundGameAmount(snapshot.DGBalance),
	}

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

	return s.store.CommitExchange(ctx, plan)
}

func (s *GameHallService) Play(ctx context.Context, input GamePlayInput) (*GamePlayResult, error) {
	if err := s.ensureEnabled(ctx); err != nil {
		return nil, err
	}
	if input.GameType != GameTypeSlots {
		return nil, ErrGameInvalidType
	}

	betAmount := roundGameAmount(input.BetAmount)
	if betAmount <= 0 || math.IsNaN(betAmount) || math.IsInf(betAmount, 0) {
		return nil, ErrGameInvalidBetAmount
	}
	key, err := NormalizeIdempotencyKey(input.IdempotencyKey)
	if err != nil {
		return nil, err
	}

	snapshot, err := s.store.GetSnapshot(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("get game hall snapshot: %w", err)
	}
	if snapshot.DGBalance < betAmount {
		return nil, ErrGameInsufficientDGBalance
	}

	multiplier, symbols, message := s.rollSlot()
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
	if s == nil || s.settings == nil {
		return ErrGameHallDisabled
	}
	values, err := s.settings.GetMultiple(ctx, []string{SettingKeyGameHallEnabled})
	if err != nil {
		return err
	}
	if values[SettingKeyGameHallEnabled] != "true" {
		return ErrGameHallDisabled
	}
	return nil
}

func defaultGameInfos() []GameInfo {
	return []GameInfo{
		{
			Type:        GameTypeSlots,
			Name:        "Slots",
			Description: "Three reels with instant DG settlement.",
			MinBet:      0.01,
			MaxBet:      1000000,
			Multipliers: []float64{0, 1.2, 3, 5, 8, 12, 18, 30, 50},
		},
	}
}

func roundGameAmount(value float64) float64 {
	return math.Round(value*1e8) / 1e8
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

func defaultSlotRoller() (float64, []string, string) {
	return rollSlotWithIntN(slotRandomIntN)
}

func rollSlotWithIntN(intN func(int) int) (float64, []string, string) {
	symbols := make([]string, 3)
	selected := make([]slotSymbolSpec, 3)

	for index := range 3 {
		symbol := pickWeightedSlotSymbol(intN)
		selected[index] = symbol
		symbols[index] = symbol.id
	}

	if selected[0].id == selected[1].id && selected[1].id == selected[2].id {
		return selected[0].payout3, symbols, "中奖"
	}

	return 0, symbols, "未中奖"
}

func pickWeightedSlotSymbol(intN func(int) int) slotSymbolSpec {
	roll := intN(slotTotalWeight)
	cumulative := 0

	for _, symbol := range slotSymbolTable {
		cumulative += symbol.weight
		if roll < cumulative {
			return symbol
		}
	}

	return slotSymbolTable[len(slotSymbolTable)-1]
}

func sumSlotWeights(symbols []slotSymbolSpec) int {
	total := 0
	for _, symbol := range symbols {
		total += symbol.weight
	}
	return total
}
