package service

import (
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"strconv"
	"strings"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/shopspring/decimal"
)

const (
	GameTypeSlots = "slots"
	GameTypeTrain = "train"
	GameTypeTexas = "texas"

	gameMinBet = 0.01
	gameMaxBet = 100000000
)

var (
	ErrGameInvalidType         = infraerrors.BadRequest("GAME_INVALID_TYPE", "game type is invalid")
	ErrGameInvalidBetAmount    = infraerrors.BadRequest("GAME_INVALID_BET_AMOUNT", "bet amount must be greater than 0")
	ErrGameInsufficientBalance = infraerrors.BadRequest("GAME_INSUFFICIENT_BALANCE", "insufficient balance")
	ErrGameBetAmountOutOfRange = infraerrors.BadRequest("GAME_BET_AMOUNT_OUT_OF_RANGE", "bet amount is out of range")
	ErrGameHallDisabled        = infraerrors.Forbidden("GAME_HALL_DISABLED", "game hall is disabled")
)

type GameInfo struct {
	Type        string    `json:"type"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	MinBet      float64   `json:"min_bet"`
	MaxBet      float64   `json:"max_bet"`
	Multipliers []float64 `json:"multipliers"`
}

type GameHallStatus struct {
	Balance float64    `json:"balance"`
	Games   []GameInfo `json:"games"`
}

type GamePlayInput struct {
	UserID         int64
	GameType       string
	BetAmount      float64
	IdempotencyKey string
	RequestID      string
}

type GamePlayResult struct {
	GameType      string   `json:"game_type"`
	BetAmount     float64  `json:"bet_amount"`
	PayoutAmount  float64  `json:"payout_amount"`
	NetAmount     float64  `json:"net_amount"`
	Multiplier    float64  `json:"multiplier"`
	BalanceBefore float64  `json:"balance_before"`
	BalanceAfter  float64  `json:"balance_after"`
	Outcome       string   `json:"outcome"`
	Symbols       []string `json:"symbols,omitempty"`
	Message       string   `json:"message"`
}

type GameService struct {
	entClient            *dbent.Client
	settingRepo          SettingRepository
	billingCacheService  *BillingCacheService
	authCacheInvalidator APIKeyAuthCacheInvalidator
}

func NewGameService(
	entClient *dbent.Client,
	settingRepo SettingRepository,
	billingCacheService *BillingCacheService,
	authCacheInvalidator APIKeyAuthCacheInvalidator,
) *GameService {
	return &GameService{
		entClient:            entClient,
		settingRepo:          settingRepo,
		billingCacheService:  billingCacheService,
		authCacheInvalidator: authCacheInvalidator,
	}
}

func (s *GameService) GetHallStatus(ctx context.Context, userID int64) (*GameHallStatus, error) {
	if err := s.ensureEnabled(ctx); err != nil {
		return nil, err
	}
	account, err := NewBankService(s.entClient).GetAccountView(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &GameHallStatus{Balance: roundMoney(account.Balance.InexactFloat64()), Games: gameInfos()}, nil
}

func (s *GameService) Play(ctx context.Context, input GamePlayInput) (*GamePlayResult, error) {
	if err := s.ensureEnabled(ctx); err != nil {
		return nil, err
	}
	gameType := strings.TrimSpace(input.GameType)
	if !isValidGameType(gameType) {
		return nil, ErrGameInvalidType
	}
	if input.BetAmount <= 0 || math.IsNaN(input.BetAmount) || math.IsInf(input.BetAmount, 0) {
		return nil, ErrGameInvalidBetAmount
	}
	bet := decimal.NewFromFloat(roundMoney(input.BetAmount)).RoundBank(18)
	if bet.LessThan(decimal.NewFromFloat(gameMinBet)) || bet.GreaterThan(decimal.NewFromFloat(gameMaxBet)) {
		return nil, ErrGameBetAmountOutOfRange
	}
	idempotencyKey, err := NormalizeIdempotencyKey(input.IdempotencyKey)
	if err != nil {
		return nil, err
	}
	if idempotencyKey == "" {
		return nil, ErrBankIdempotencyKeyRequired
	}

	account, err := NewBankService(s.entClient).GetAccountView(ctx, input.UserID)
	if err != nil {
		return nil, err
	}
	if account.Status != BankAccountStatusActive {
		return nil, ErrBankAccountNotActive
	}
	if account.Balance.LessThan(bet) {
		return nil, ErrGameInsufficientBalance
	}

	multiplier, symbols := rollGame(gameType)
	payout := bet.Mul(decimal.NewFromFloat(multiplier)).RoundBank(18)
	requests := []TransferFundsRequest{
		gameTransferRequest(input, gameType, bet, BankTxTypeSlotBet, "game bet", "bet"),
	}
	if payout.GreaterThan(decimal.Zero) {
		requests = append(requests, gameTransferRequest(input, gameType, payout, BankTxTypeSlotWin, "game payout", "win"))
	}
	results, err := NewBankService(s.entClient).TransferFundsBatch(ctx, requests)
	if err != nil {
		return nil, err
	}
	s.invalidateBalanceCaches(ctx, input.UserID)

	balanceAfter := results[len(results)-1].Balance
	net := payout.Sub(bet).RoundBank(18)
	return &GamePlayResult{
		GameType:      gameType,
		BetAmount:     roundMoney(bet.InexactFloat64()),
		PayoutAmount:  roundMoney(payout.InexactFloat64()),
		NetAmount:     roundMoney(net.InexactFloat64()),
		Multiplier:    multiplier,
		BalanceBefore: roundMoney(account.Balance.InexactFloat64()),
		BalanceAfter:  roundMoney(balanceAfter.InexactFloat64()),
		Outcome:       gameOutcome(net),
		Symbols:       symbols,
		Message:       gameMessage(gameType, net, multiplier),
	}, nil
}

func gameTransferRequest(input GamePlayInput, gameType string, amount decimal.Decimal, txType, description, phase string) TransferFundsRequest {
	return TransferFundsRequest{
		UserID:           input.UserID,
		Amount:           amount,
		Type:             txType,
		BusinessModule:   BankBusinessModuleGame,
		Description:      description,
		IdempotencyScope: fmt.Sprintf("game:%d:%s", input.UserID, phase),
		IdempotencyKey:   fmt.Sprintf("%s:%s", input.IdempotencyKey, phase),
		ReferenceType:    "game_round",
		ReferenceID:      input.IdempotencyKey,
		RequestID:        input.RequestID,
		Metadata: map[string]any{
			"game_type": gameType,
			"phase":     phase,
		},
	}
}

func (s *GameService) ensureEnabled(ctx context.Context) error {
	if s.settingRepo == nil {
		return ErrGameHallDisabled
	}
	settings, err := s.settingRepo.GetMultiple(ctx, []string{SettingKeyGameHallEnabled})
	if err != nil {
		return err
	}
	if settings[SettingKeyGameHallEnabled] != "true" {
		return ErrGameHallDisabled
	}
	return nil
}

func (s *GameService) invalidateBalanceCaches(ctx context.Context, userID int64) {
	if s.authCacheInvalidator != nil {
		s.authCacheInvalidator.InvalidateAuthCacheByUserID(ctx, userID)
	}
	if s.billingCacheService != nil {
		_ = s.billingCacheService.InvalidateUserBalance(ctx, userID)
	}
}

func gameInfos() []GameInfo {
	return []GameInfo{
		{Type: GameTypeSlots, Name: "Slots", Description: "Three reels with instant settlement.", MinBet: gameMinBet, MaxBet: gameMaxBet, Multipliers: []float64{0, 1.2, 3, 5, 8, 12, 18, 30, 50}},
		{Type: GameTypeTrain, Name: "Train", Description: "A short risk run with larger payouts on rare stops.", MinBet: gameMinBet, MaxBet: gameMaxBet, Multipliers: []float64{0, 0.5, 1.2, 2, 5}},
		{Type: GameTypeTexas, Name: "Texas", Description: "A simplified poker hand simulator.", MinBet: gameMinBet, MaxBet: gameMaxBet, Multipliers: []float64{0, 0.8, 1.5, 3, 8}},
	}
}

func isValidGameType(gameType string) bool {
	switch gameType {
	case GameTypeSlots, GameTypeTrain, GameTypeTexas:
		return true
	default:
		return false
	}
}

func rollGame(gameType string) (float64, []string) {
	switch gameType {
	case GameTypeSlots:
		return rollSlots()
	case GameTypeTrain:
		return rollTrain()
	case GameTypeTexas:
		return rollTexas()
	default:
		return 0, nil
	}
}

func rollSlots() (float64, []string) {
	roll := rand.Float64()
	switch {
	case roll < 0.001:
		return 50, []string{"7", "7", "7"}
	case roll < 0.003:
		return 30, []string{"diamond", "diamond", "diamond"}
	case roll < 0.007:
		return 18, []string{"star", "star", "star"}
	case roll < 0.015:
		return 12, []string{"bell", "bell", "bell"}
	case roll < 0.029:
		return 8, []string{"grape", "grape", "grape"}
	case roll < 0.054:
		if rand.Float64() < 0.5 {
			return 5, []string{"lemon", "lemon", "lemon"}
		}
		return 5, []string{"orange", "orange", "orange"}
	case roll < 0.104:
		return 3, []string{"cherry", "cherry", "cherry"}
	case roll < 0.344:
		symbol := randomSlotSymbol()
		return 1.2, []string{symbol, randomSlotSymbolExcept(symbol), randomSlotSymbolExcept(symbol)}
	default:
		return 0, losingSlotSymbols()
	}
}

func randomSlotSymbol() string {
	symbols := []string{"cherry", "lemon", "orange", "grape", "bell", "star", "diamond", "7"}
	return symbols[rand.IntN(len(symbols))]
}

func randomSlotSymbolExcept(symbol string) string {
	for {
		next := randomSlotSymbol()
		if next != symbol {
			return next
		}
	}
}

func losingSlotSymbols() []string {
	for {
		symbols := []string{randomSlotSymbol(), randomSlotSymbol(), randomSlotSymbol()}
		if symbols[0] != symbols[1] && symbols[1] != symbols[2] && symbols[0] != symbols[2] {
			return symbols
		}
	}
}

func rollTrain() (float64, []string) {
	roll := rand.Float64()
	switch {
	case roll < 0.08:
		return 5, []string{"engine", "car", "car", "finish"}
	case roll < 0.22:
		return 2, []string{"engine", "car", "station"}
	case roll < 0.48:
		return 1.2, []string{"engine", "car"}
	case roll < 0.74:
		return 0.5, []string{"engine", "switch"}
	default:
		return 0, []string{"engine", "crash"}
	}
}

func rollTexas() (float64, []string) {
	roll := rand.Float64()
	switch {
	case roll < 0.04:
		return 8, []string{"A-spade", "K-spade", "Q-spade", "J-spade", "10-spade"}
	case roll < 0.14:
		return 3, []string{"Q-heart", "Q-spade", "Q-club", "7-diamond", "2-club"}
	case roll < 0.34:
		return 1.5, []string{"10-heart", "10-club", "8-spade", "8-diamond", "3-club"}
	case roll < 0.62:
		return 0.8, []string{"A-heart", "K-club", "9-spade", "6-diamond", "2-heart"}
	default:
		return 0, []string{"9-club", "7-heart", "5-spade", "3-diamond", "2-club"}
	}
}

func gameOutcome(net decimal.Decimal) string {
	switch {
	case net.IsPositive():
		return "win"
	case net.IsNegative():
		return "lose"
	default:
		return "draw"
	}
}

func gameMessage(gameType string, net decimal.Decimal, multiplier float64) string {
	if net.IsPositive() {
		return "Win: payout is " + formatGameMultiplier(multiplier) + "x"
	}
	if net.IsZero() {
		return "Draw: bet returned"
	}
	switch gameType {
	case GameTypeTrain:
		return "Train stopped short"
	case GameTypeTexas:
		return "Hand did not qualify"
	default:
		return "No winning combination"
	}
}

func formatGameMultiplier(value float64) string {
	return strings.TrimRight(strings.TrimRight(strconv.FormatFloat(value, 'f', 2, 64), "0"), ".")
}

func roundMoney(value float64) float64 {
	return math.Round(value*100000000) / 100000000
}
