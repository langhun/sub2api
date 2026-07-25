package walletextension

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/Wei-Shaw/sub2api/internal/custom/modules/wallet-extension/contract"
	"github.com/Wei-Shaw/sub2api/internal/custom/platform"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	directTransferReceiverSearchLimit = 8
	accountStatusActive               = "active"
)

var (
	ErrTransferDisabled             = infraerrors.Forbidden("TRANSFER_DISABLED", "transfer feature is disabled")
	ErrTransferSelf                 = infraerrors.BadRequest("TRANSFER_SELF", "cannot transfer to yourself")
	ErrTransferAmountInvalid        = infraerrors.BadRequest("TRANSFER_AMOUNT_INVALID", "invalid transfer amount")
	ErrTransferInsufficient         = infraerrors.BadRequest("TRANSFER_INSUFFICIENT", "insufficient balance")
	ErrTransferDailyLimit           = infraerrors.Forbidden("TRANSFER_DAILY_LIMIT", "daily transfer limit exceeded")
	ErrTransferDailyCount           = infraerrors.Forbidden("TRANSFER_DAILY_COUNT", "daily transfer count limit exceeded")
	ErrTransferReceiverNotFound     = infraerrors.NotFound("RECEIVER_NOT_FOUND", "receiver not found")
	ErrTransferReceiverQueryInvalid = infraerrors.BadRequest("RECEIVER_QUERY_INVALID", "receiver query must be a positive user ID or at least 2 characters")
	ErrTransferReceiverAmbiguous    = infraerrors.Conflict("RECEIVER_AMBIGUOUS", "receiver query matches multiple users; use a user ID or exact email")
	ErrTransferLeaderboardDisabled  = infraerrors.NotFound("TRANSFER_LEADERBOARD_DISABLED", "transfer leaderboard is disabled")
	ErrTransferNotFound             = infraerrors.NotFound("TRANSFER_NOT_FOUND", "transfer not found")
	ErrTransferAlreadyFrozen        = infraerrors.BadRequest("TRANSFER_ALREADY_FROZEN", "transfer already frozen")
	ErrTransferAlreadyRevoked       = infraerrors.BadRequest("TRANSFER_ALREADY_REVOKED", "transfer already revoked")
)

// DirectTransferCommitPlan is the fully validated atomic write requested from the repository.
type DirectTransferCommitPlan struct {
	SenderID        int64
	ReceiverID      int64
	Amount          float64
	Fee             float64
	FeeRate         float64
	GrossAmount     float64
	Memo            *string
	DailyLimit      float64
	DailyCountLimit int
}

// Service implements the direct-transfer slice through wallet-owned ports.
type Service struct {
	repository    DirectTransferRepository
	settings      contract.SettingsReader
	accounts      contract.AccountReader
	recipients    contract.RecipientResolver
	subscriptions contract.ActiveSubscriptionReader
	balances      contract.BalanceWriter
	cache         contract.BalanceCacheInvalidator
}

// NewService constructs the direct-transfer service from narrow platform adapters.
func NewService(
	repository DirectTransferRepository,
	settings contract.SettingsReader,
	accounts contract.AccountReader,
	recipients contract.RecipientResolver,
	subscriptions contract.ActiveSubscriptionReader,
	balances contract.BalanceWriter,
	cache contract.BalanceCacheInvalidator,
) *Service {
	return &Service{
		repository: repository, settings: settings, accounts: accounts, recipients: recipients,
		subscriptions: subscriptions, balances: balances, cache: cache,
	}
}

// Transfer validates and commits one direct transfer.
func (s *Service) Transfer(ctx context.Context, senderID int64, request DirectTransferRequest) (DirectTransferRecord, error) {
	settings, err := s.directTransferSettings(ctx)
	if err != nil {
		return DirectTransferRecord{}, err
	}
	if err := validateDirectTransfer(senderID, request.ReceiverID, request.Amount, settings); err != nil {
		return DirectTransferRecord{}, err
	}
	if _, err := s.activeRecipient(ctx, request.ReceiverID); err != nil {
		return DirectTransferRecord{}, err
	}
	feeRate := s.feeRate(ctx, senderID, settings)
	amount := roundTransferAmount(request.Amount)
	fee := math.Max(0, roundTransferAmount(amount*feeRate))
	gross := roundTransferAmount(amount + fee)

	record, err := s.repository.CommitDirectTransfer(ctx, DirectTransferCommitPlan{
		SenderID: senderID, ReceiverID: request.ReceiverID, Amount: amount, Fee: fee, FeeRate: feeRate,
		GrossAmount: gross, Memo: request.Memo, DailyLimit: settings.DailyLimit, DailyCountLimit: settings.DailyCountLimit,
	})
	if err != nil {
		return DirectTransferRecord{}, err
	}
	s.invalidateBalances(ctx, senderID, request.ReceiverID)
	return record, nil
}

// Preview returns the effective direct-transfer fee and limit state without writing balance.
func (s *Service) Preview(ctx context.Context, senderID, receiverID int64, amount float64) (DirectTransferPreview, error) {
	settings, err := s.directTransferSettings(ctx)
	if err != nil {
		return DirectTransferPreview{}, err
	}
	if err := validateDirectTransfer(senderID, receiverID, amount, settings); err != nil {
		return DirectTransferPreview{}, err
	}
	recipient, err := s.activeRecipient(ctx, receiverID)
	if err != nil {
		return DirectTransferPreview{}, err
	}
	dayStart := startOfLocalDay()
	dailyTotal, dailyCount, err := s.repository.GetDirectTransferDailyUsage(ctx, senderID, dayStart, dayStart.AddDate(0, 0, 1))
	if err != nil {
		return DirectTransferPreview{}, fmt.Errorf("check direct transfer daily limit: %w", err)
	}
	feeRate := s.feeRate(ctx, senderID, settings)
	amount = roundTransferAmount(amount)
	fee := math.Max(0, roundTransferAmount(amount*feeRate))
	gross := roundTransferAmount(amount + fee)
	if settings.DailyLimit > 0 && dailyTotal+gross > settings.DailyLimit {
		return DirectTransferPreview{}, ErrTransferDailyLimit
	}
	if settings.DailyCountLimit > 0 && dailyCount >= settings.DailyCountLimit {
		return DirectTransferPreview{}, ErrTransferDailyCount
	}
	preview := DirectTransferPreview{Fee: fee, FeeRate: feeRate, GrossAmount: gross, Receiver: recipient, ReceiverID: receiverID, ReceiverDisplay: recipient.DisplayName}
	if settings.DailyLimit > 0 {
		preview.DailyRemainingAmount = math.Max(0, roundTransferAmount(settings.DailyLimit-dailyTotal))
	}
	if settings.DailyCountLimit > 0 {
		preview.DailyRemainingCount = max(0, settings.DailyCountLimit-dailyCount)
	}
	return preview, nil
}

// ResolveRecipient finds one eligible direct-transfer recipient.
func (s *Service) ResolveRecipient(ctx context.Context, requesterID int64, rawQuery string) (contract.Recipient, error) {
	if _, err := s.enabledSettings(ctx); err != nil {
		return contract.Recipient{}, err
	}
	query := strings.TrimSpace(rawQuery)
	if query != "" {
		if id, err := strconv.ParseInt(query, 10, 64); err == nil {
			if id <= 0 {
				return contract.Recipient{}, ErrTransferReceiverQueryInvalid
			}
		} else if isASCIIDigits(query) {
			return contract.Recipient{}, ErrTransferReceiverQueryInvalid
		}
	}
	if !isASCIIDigits(query) && utf8.RuneCountInString(query) < 2 {
		return contract.Recipient{}, ErrTransferReceiverQueryInvalid
	}
	if s.recipients == nil {
		return contract.Recipient{}, ErrTransferReceiverNotFound
	}
	return s.recipients.ResolveDirectTransferRecipient(ctx, requesterID, query)
}

// SearchRecipients returns eligible recipient hints without exposing raw identities.
func (s *Service) SearchRecipients(ctx context.Context, requesterID int64, rawQuery string) ([]contract.RecipientCandidate, error) {
	if _, err := s.enabledSettings(ctx); err != nil {
		return nil, err
	}
	query := strings.TrimSpace(rawQuery)
	if utf8.RuneCountInString(query) < 2 && !isASCIIDigits(query) {
		return nil, ErrTransferReceiverQueryInvalid
	}
	if s.recipients == nil {
		return []contract.RecipientCandidate{}, nil
	}
	return s.recipients.SearchDirectTransferRecipients(ctx, requesterID, query, directTransferReceiverSearchLimit)
}

// ListHistory returns an account's direct-transfer history.
func (s *Service) ListHistory(ctx context.Context, query DirectTransferHistoryQuery) ([]DirectTransferRecord, int, error) {
	if _, err := s.enabledSettings(ctx); err != nil {
		return nil, 0, err
	}
	return s.repository.ListDirectTransferHistory(ctx, query)
}

// GetStats returns an account's direct-transfer aggregates.
func (s *Service) GetStats(ctx context.Context, accountID int64) (DirectTransferStats, error) {
	if _, err := s.enabledSettings(ctx); err != nil {
		return DirectTransferStats{}, err
	}
	return s.repository.GetDirectTransferStats(ctx, accountID)
}

func (s *Service) activeRecipient(ctx context.Context, accountID int64) (contract.Recipient, error) {
	if s.accounts == nil {
		return contract.Recipient{}, ErrTransferReceiverNotFound
	}
	account, err := s.accounts.GetAccount(ctx, accountID)
	if err != nil || account.ID == 0 || account.Status != accountStatusActive {
		return contract.Recipient{}, ErrTransferReceiverNotFound
	}
	return recipientFromAccount(account), nil
}

func (s *Service) directTransferSettings(ctx context.Context) (contract.DirectTransferSettings, error) {
	if s.settings == nil {
		return contract.DirectTransferSettings{}, ErrTransferDisabled
	}
	settings, err := s.settings.GetWalletExtensionSettings(ctx)
	if err != nil {
		return contract.DirectTransferSettings{}, err
	}
	if !settings.DirectTransfer.Enabled {
		return contract.DirectTransferSettings{}, ErrTransferDisabled
	}
	return settings.DirectTransfer, nil
}

func (s *Service) enabledSettings(ctx context.Context) (contract.DirectTransferSettings, error) {
	return s.directTransferSettings(ctx)
}

func (s *Service) feeRate(ctx context.Context, senderID int64, settings contract.DirectTransferSettings) float64 {
	if !settings.VIPFeeExempt || s.subscriptions == nil {
		return settings.FeeRate
	}
	hasActive, err := s.subscriptions.HasActiveSubscription(ctx, senderID)
	if err == nil && hasActive {
		return 0
	}
	return settings.FeeRate
}

func (s *Service) invalidateBalances(ctx context.Context, senderID, receiverID int64) {
	if s.cache == nil {
		return
	}
	_ = s.cache.InvalidateBalance(ctx, senderID)
	if receiverID != senderID {
		_ = s.cache.InvalidateBalance(ctx, receiverID)
	}
}

func validateDirectTransfer(senderID, receiverID int64, amount float64, settings contract.DirectTransferSettings) error {
	if senderID == receiverID {
		return ErrTransferSelf
	}
	amount = roundTransferAmount(amount)
	if math.IsNaN(amount) || math.IsInf(amount, 0) || amount <= 0 || amount < settings.MinimumAmount || (settings.MaximumAmount > 0 && amount > settings.MaximumAmount) {
		return ErrTransferAmountInvalid
	}
	return nil
}

func roundTransferAmount(amount float64) float64 { return math.Round(amount*1e8) / 1e8 }

func recipientFromAccount(account contract.Account) contract.Recipient {
	display := strings.TrimSpace(account.Username)
	if display == "" {
		display = maskRecipientEmail(account.Email)
	}
	if display == "" {
		display = platform.UserDisplayName("", "", account.ID)
	}
	return contract.Recipient{Account: account, DisplayName: display}
}

func isASCIIDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func maskRecipientIdentity(value string) string {
	runes := []rune(strings.TrimSpace(value))
	switch len(runes) {
	case 0:
		return ""
	case 1:
		return string(runes[0])
	case 2:
		return string(runes[0]) + "*"
	default:
		return string(runes[0]) + strings.Repeat("*", len(runes)-2) + string(runes[len(runes)-1])
	}
}

func maskRecipientEmail(value string) string {
	email := strings.TrimSpace(value)
	local, domain, ok := strings.Cut(email, "@")
	if !ok || local == "" || domain == "" {
		return maskRecipientIdentity(email)
	}
	parts := strings.Split(domain, ".")
	for i := 0; i < len(parts)-1; i++ {
		parts[i] = maskRecipientIdentity(parts[i])
	}
	return maskRecipientIdentity(local) + "@" + strings.Join(parts, ".")
}
