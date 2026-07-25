package walletextension

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/custom/modules/wallet-extension/contract"
)

const (
	// DirectTransferType is the only transfer kind owned by this module boundary.
	DirectTransferType = "direct"

	legacyDirectTransferHandlerPath    = "backend/internal/handler/balance_transfer_handler.go"
	legacyDirectTransferServicePath    = "backend/internal/service/balance_transfer_service.go"
	legacyDirectTransferRepositoryPath = "backend/internal/repository/balance_transfer_repo.go"
)

// DirectTransferRequest is the transport-neutral input for a direct transfer.
type DirectTransferRequest struct {
	ReceiverID     int64
	Amount         float64
	Memo           *string
	IdempotencyKey string
}

// DirectTransferPreview describes the fee and daily-limit result before a transfer commits.
type DirectTransferPreview struct {
	Fee                  float64            `json:"fee"`
	FeeRate              float64            `json:"fee_rate"`
	GrossAmount          float64            `json:"gross_amount"`
	Receiver             contract.Recipient `json:"-"`
	ReceiverID           int64              `json:"receiver_id"`
	ReceiverDisplay      string             `json:"receiver_display"`
	DailyRemainingAmount float64            `json:"daily_remaining_amount"`
	DailyRemainingCount  int                `json:"daily_remaining_count"`
}

// DirectTransferRecord is the direct-transfer projection shared across migration layers.
type DirectTransferRecord struct {
	ID              int64      `json:"id"`
	SenderID        int64      `json:"sender_id"`
	ReceiverID      int64      `json:"receiver_id"`
	SenderDisplay   string     `json:"sender_display"`
	ReceiverDisplay string     `json:"receiver_display"`
	Amount          float64    `json:"amount"`
	Fee             float64    `json:"fee"`
	FeeRate         float64    `json:"fee_rate"`
	GrossAmount     float64    `json:"gross_amount"`
	TransferType    string     `json:"transfer_type"`
	Status          string     `json:"status"`
	Memo            *string    `json:"memo"`
	FrozenAt        *time.Time `json:"frozen_at"`
	FrozenBy        *int64     `json:"frozen_by"`
	RevokeReason    *string    `json:"revoke_reason"`
	CreatedAt       time.Time  `json:"created_at"`
}

// DirectTransferHistoryQuery scopes a paginated user transfer history query.
type DirectTransferHistoryQuery struct {
	AccountID int64
	Role      string
	Page      int
	PageSize  int
}

// DirectTransferStats is the user-facing aggregate for completed direct transfers.
type DirectTransferStats struct {
	TotalSent     float64 `json:"total_sent"`
	TotalReceived float64 `json:"total_received"`
	TotalFeePaid  float64 `json:"total_fee_paid"`
}

// DirectTransferService is the target service contract for the direct-transfer slice.
// It intentionally excludes red-packet, blind-box, and leaderboard behavior.
type DirectTransferService interface {
	Transfer(ctx context.Context, senderID int64, request DirectTransferRequest) (DirectTransferRecord, error)
	Preview(ctx context.Context, senderID int64, receiverID int64, amount float64) (DirectTransferPreview, error)
	ResolveRecipient(ctx context.Context, requesterID int64, query string) (contract.Recipient, error)
	SearchRecipients(ctx context.Context, requesterID int64, query string) ([]contract.RecipientCandidate, error)
	ListHistory(ctx context.Context, query DirectTransferHistoryQuery) ([]DirectTransferRecord, int, error)
	GetStats(ctx context.Context, accountID int64) (DirectTransferStats, error)
}

// DirectTransferHandler is the target transport adapter contract. HTTP binding,
// authentication, and idempotency middleware remain outside this module until route migration.
type DirectTransferHandler interface {
	HandleTransfer(ctx context.Context, senderID int64, request DirectTransferRequest) (DirectTransferRecord, error)
	HandlePreview(ctx context.Context, senderID int64, receiverID int64, amount float64) (DirectTransferPreview, error)
	HandleResolveRecipient(ctx context.Context, requesterID int64, query string) (contract.Recipient, error)
	HandleSearchRecipients(ctx context.Context, requesterID int64, query string) ([]contract.RecipientCandidate, error)
	HandleHistory(ctx context.Context, query DirectTransferHistoryQuery) ([]DirectTransferRecord, int, error)
	HandleStats(ctx context.Context, accountID int64) (DirectTransferStats, error)
}

// DirectTransferRepository is the target persistence contract for direct transfers only.
type DirectTransferRepository interface {
	CommitDirectTransfer(ctx context.Context, plan DirectTransferCommitPlan) (DirectTransferRecord, error)
	CreateDirectTransfer(ctx context.Context, record *DirectTransferRecord) error
	GetDirectTransfer(ctx context.Context, transferID int64) (DirectTransferRecord, error)
	ListDirectTransferHistory(ctx context.Context, query DirectTransferHistoryQuery) ([]DirectTransferRecord, int, error)
	GetDirectTransferDailyUsage(ctx context.Context, senderID int64, start, end time.Time) (amount float64, count int, err error)
	GetDirectTransferStats(ctx context.Context, accountID int64) (DirectTransferStats, error)
}

// MigrationLayer identifies one legacy layer being detached from the shared feature implementation.
type MigrationLayer string

const (
	MigrationLayerRepository MigrationLayer = "repository"
	MigrationLayerService    MigrationLayer = "service"
	MigrationLayerHandler    MigrationLayer = "handler"
)

// MigrationStep identifies one bounded layer migration and its legacy source.
type MigrationStep struct {
	Layer        MigrationLayer
	LegacySource string
	Target       string
	Scope        []string
}

// MigrationPlan documents the direct-transfer extraction order without changing runtime wiring.
type MigrationPlan struct {
	Name                 string
	IncludedCapabilities []string
	ExcludedCapabilities []string
	Steps                []MigrationStep
}

// DirectTransferMigrationPlan records the complete transfer-ledger extraction
// owned by wallet-extension. Red-packet and blind-box lifecycles remain activity-owned.
var DirectTransferMigrationPlan = MigrationPlan{
	Name:                 "wallet-extension-transfer-ledger",
	IncludedCapabilities: []string{"recipient resolution", "preview", "point-to-point transfer", "history", "user stats", "transfer leaderboard", "transfer administration", "batch distribution"},
	ExcludedCapabilities: []string{"red packet", "blind box"},
	Steps: []MigrationStep{
		{
			Layer:        MigrationLayerRepository,
			LegacySource: legacyDirectTransferRepositoryPath,
			Target:       "backend/internal/custom/modules/wallet-extension/direct_transfer_repository.go",
			Scope:        []string{"transfer ledger", "daily usage", "history", "user stats", "leaderboard", "administrative queries"},
		},
		{
			Layer:        MigrationLayerService,
			LegacySource: legacyDirectTransferServicePath,
			Target:       "backend/internal/custom/modules/wallet-extension/direct_transfer_service.go",
			Scope:        []string{"validation", "fee calculation", "atomic debit and credit", "revocation", "batch distribution"},
		},
		{
			Layer:        MigrationLayerHandler,
			LegacySource: legacyDirectTransferHandlerPath,
			Target:       "backend/internal/custom/modules/wallet-extension/direct_transfer_handler.go",
			Scope:        []string{"transfer", "preview", "recipient lookup", "history", "stats", "leaderboard", "administration"},
		},
	},
}
