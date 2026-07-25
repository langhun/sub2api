// Package redpacket owns the Activity red-packet module boundary.
package redpacket

import (
	"context"
	"time"
)

// Type controls how a red packet is split between claimants.
type Type string

const (
	TypeEqual  Type = "equal"
	TypeRandom Type = "random"
)

// Status describes the lifecycle state of a red packet.
type Status string

const (
	StatusActive    Status = "active"
	StatusExhausted Status = "exhausted"
	StatusExpired   Status = "expired"
)

// RedPacket is the Activity-owned view of an issued red packet.
// Amounts retain the existing eight-decimal accounting precision.
type RedPacket struct {
	ID              int64     `json:"id"`
	SenderID        int64     `json:"sender_id"`
	TotalAmount     float64   `json:"total_amount"`
	TotalCount      int       `json:"total_count"`
	RemainingAmount float64   `json:"remaining_amount"`
	RemainingCount  int       `json:"remaining_count"`
	Type            Type      `json:"redpacket_type"`
	Fee             float64   `json:"fee"`
	FeeRate         float64   `json:"fee_rate"`
	Code            string    `json:"code"`
	Status          Status    `json:"status"`
	Memo            *string   `json:"memo"`
	ExpiresAt       time.Time `json:"expire_at"`
	CreatedAt       time.Time `json:"created_at"`
}

// Claim records one successful red-packet claim.
type Claim struct {
	ID          int64     `json:"id"`
	RedPacketID int64     `json:"redpacket_id"`
	UserID      int64     `json:"user_id"`
	UserDisplay string    `json:"user_display"`
	Amount      float64   `json:"amount"`
	AuditID     *int64    `json:"transfer_id"`
	CreatedAt   time.Time `json:"created_at"`
}

// CreateRequest contains the caller-owned input for creating a red packet.
type CreateRequest struct {
	SenderID       int64
	TotalAmount    float64
	Count          int
	Type           Type
	Memo           *string
	IdempotencyKey string
}

// ClaimRequest contains the caller-owned input for claiming a red packet.
type ClaimRequest struct {
	UserID         int64
	Code           string
	IdempotencyKey string
}

// ExpiryRefund identifies one expired red packet whose unclaimed balance was
// returned to its sender. A zero amount means another transaction already
// exhausted or refunded the packet.
type ExpiryRefund struct {
	RedPacketID    int64
	SenderID       int64
	ReturnedAmount float64
	OccurredAt     time.Time
}

// ExpiryRunResult is the aggregate outcome of one expiry-worker cycle.
type ExpiryRunResult struct {
	Processed int
	Refunds   []ExpiryRefund
}

// Creator is the module-owned create boundary replacing
// BalanceTransferService.CreateRedPacket.
type Creator interface {
	Create(ctx context.Context, request CreateRequest) (*RedPacket, error)
}

// Claimer is the module-owned claim boundary replacing
// BalanceTransferService.ClaimRedPacket.
type Claimer interface {
	Claim(ctx context.Context, request ClaimRequest) (*Claim, error)
}

// ExpiryRefunder is the module-owned expiry boundary replacing
// BalanceTransferService.ExpireRedPackets.
type ExpiryRefunder interface {
	RefundExpired(ctx context.Context) (ExpiryRunResult, error)
}

// QueryService collects the read operations that must leave
// BalanceTransferService with the mutation boundaries.
type QueryService interface {
	Get(ctx context.Context, redPacketID int64) (*RedPacket, error)
	GetForParticipant(ctx context.Context, requesterID, redPacketID int64) (*RedPacket, []Claim, error)
	ListCreatedBy(ctx context.Context, senderID int64, page, pageSize int) ([]RedPacket, int, error)
	ListClaimedBy(ctx context.Context, userID int64, page, pageSize int) ([]RedPacket, int, error)
	ListAll(ctx context.Context, page, pageSize int) ([]RedPacket, int, error)
}

// Service is the complete public Activity red-packet surface.
type Service interface {
	Creator
	Claimer
	ExpiryRefunder
	QueryService
}
