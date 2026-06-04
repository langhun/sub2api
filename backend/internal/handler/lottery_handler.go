package handler

import (
	"context"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/shopspring/decimal"

	"github.com/gin-gonic/gin"
)

const defaultLotteryType = service.LotteryTypeSSQ

type lotteryService interface {
	CreateBet(ctx context.Context, input service.LotteryBetInput) (*service.LotteryBetResult, error)
	GetCurrentIssue(ctx context.Context, lotteryType string) (*service.Issue, error)
	GetMyOrders(ctx context.Context, query service.LotteryOrderQuery) ([]service.LotteryOrderView, error)
	GetJackpot(ctx context.Context, lotteryType string) (*service.LotteryJackpotView, error)
}

type LotteryHandler struct {
	lotteryService lotteryService
}

func NewLotteryHandler(lotteryService *service.LotteryService) *LotteryHandler {
	return &LotteryHandler{lotteryService: lotteryService}
}

type lotteryCurrentResponse struct {
	IssueNo        string    `json:"issue_no"`
	LotteryType    string    `json:"lottery_type"`
	OpenTime       time.Time `json:"open_time"`
	CutoffTime     time.Time `json:"cutoff_time"`
	IsClosed       bool      `json:"is_closed"`
	JackpotBalance string    `json:"jackpot_balance"`
}

type lotteryBetRequest struct {
	RedBalls []int `json:"red_balls" binding:"required"`
	BlueBall int   `json:"blue_ball" binding:"required"`
}

type lotteryBetResponse struct {
	OrderID     int64     `json:"order_id,omitempty"`
	OrderIDs    []int64   `json:"order_ids,omitempty"`
	IssueNo     string    `json:"issue_no"`
	LotteryType string    `json:"lottery_type"`
	Cost        string    `json:"cost"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
}

type lotteryOrderResponse struct {
	OrderID     int64     `json:"order_id"`
	IssueNo     string    `json:"issue_no"`
	LotteryType string    `json:"lottery_type"`
	RedBalls    []string  `json:"red_balls"`
	BlueBall    string    `json:"blue_ball"`
	Cost        string    `json:"cost"`
	Status      string    `json:"status"`
	Reward      string    `json:"reward"`
	PrizeLevel  string    `json:"prize_level"`
	CreatedAt   time.Time `json:"created_at"`
}

func (h *LotteryHandler) GetCurrent(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	issue, err := h.lotteryService.GetCurrentIssue(c.Request.Context(), defaultLotteryType)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	jackpot, err := h.lotteryService.GetJackpot(c.Request.Context(), defaultLotteryType)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, lotteryCurrentResponse{
		IssueNo:        issue.IssueNo,
		LotteryType:    issue.LotteryType,
		OpenTime:       issue.OpenTime,
		CutoffTime:     issue.CutoffTime,
		IsClosed:       issue.IsClosed,
		JackpotBalance: decimalStringFromDecimal(jackpot.Balance),
	})
}

func (h *LotteryHandler) Bet(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req lotteryBetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: red_balls and blue_ball are required")
		return
	}

	redBalls := make([]string, 0, len(req.RedBalls))
	for _, ball := range req.RedBalls {
		redBalls = append(redBalls, strconv.Itoa(ball))
	}
	blueBall := strconv.Itoa(req.BlueBall)

	executeUserIdempotentJSON(c, "user.lottery.bet", req, service.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) {
		result, err := h.lotteryService.CreateBet(ctx, service.LotteryBetInput{
			UserID:         subject.UserID,
			LotteryType:    defaultLotteryType,
			RedBalls:       redBalls,
			BlueBall:       blueBall,
			BetCount:       1,
			IdempotencyKey: c.GetHeader("Idempotency-Key"),
			RequestID:      c.GetHeader("Idempotency-Key"),
		})
		if err != nil {
			return nil, err
		}
		return lotteryBetResponseFromResult(result, time.Now()), nil
	})
}

func (h *LotteryHandler) GetOrders(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	orders, err := h.lotteryService.GetMyOrders(c.Request.Context(), service.LotteryOrderQuery{
		UserID:      subject.UserID,
		LotteryType: defaultLotteryType,
		IssueNo:     c.Query("issue_no"),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	out := make([]lotteryOrderResponse, 0, len(orders))
	for _, order := range orders {
		out = append(out, lotteryOrderResponseFromView(order))
	}
	response.Success(c, out)
}

func lotteryBetResponseFromResult(result *service.LotteryBetResult, now time.Time) lotteryBetResponse {
	if result == nil {
		return lotteryBetResponse{CreatedAt: now}
	}
	out := lotteryBetResponse{
		OrderIDs:    result.OrderIDs,
		IssueNo:     result.IssueNo,
		LotteryType: result.LotteryType,
		Cost:        decimalStringFromDecimal(result.Cost),
		Status:      result.Status,
		CreatedAt:   now,
	}
	if len(result.OrderIDs) == 1 {
		out.OrderID = result.OrderIDs[0]
	}
	return out
}

func lotteryOrderResponseFromView(order service.LotteryOrderView) lotteryOrderResponse {
	return lotteryOrderResponse{
		OrderID:     order.ID,
		IssueNo:     order.IssueNo,
		LotteryType: order.LotteryType,
		RedBalls:    order.RedBalls,
		BlueBall:    order.BlueBall,
		Cost:        decimalStringFromDecimal(order.Cost),
		Status:      order.Status,
		Reward:      decimalStringFromDecimal(order.Reward),
		PrizeLevel:  order.PrizeLevel,
		CreatedAt:   order.CreatedAt,
	}
}

func decimalStringFromDecimal(value decimal.Decimal) string {
	return value.String()
}
