package walletextension

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/Wei-Shaw/sub2api/internal/custom/modules/wallet-extension/contract"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

const directTransferReceiverSearchLimit = 8

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

type legacyRecipientResolver interface {
	ResolveActiveTransferReceiver(ctx context.Context, query string, numericID *int64) (*service.User, error)
	SearchActiveTransferReceivers(ctx context.Context, query string, requesterID int64, limit int) ([]*service.User, error)
}

type directTransferUserReader interface {
	GetByID(ctx context.Context, id int64) (*service.User, error)
}

type activeSubscriptionReader interface {
	ListActiveUserSubscriptions(ctx context.Context, userID int64) ([]service.UserSubscription, error)
}

type balanceCacheInvalidator interface {
	InvalidateUserBalance(ctx context.Context, userID int64) error
}

// SettingsAdapter narrows the legacy setting service to wallet-extension's direct-transfer policy.
type SettingsAdapter struct{ legacy *service.SettingService }

// NewSettingsAdapter adapts the shared setting service without moving its ownership.
func NewSettingsAdapter(legacy *service.SettingService) *SettingsAdapter {
	return &SettingsAdapter{legacy: legacy}
}

// GetWalletExtensionSettings returns only direct-transfer settings.
func (a *SettingsAdapter) GetWalletExtensionSettings(ctx context.Context) (contract.Settings, error) {
	if a == nil || a.legacy == nil {
		return contract.Settings{}, nil
	}
	settings, err := a.legacy.GetAllSettings(ctx)
	if err != nil {
		return contract.Settings{}, err
	}
	return contract.Settings{DirectTransfer: contract.DirectTransferSettings{
		Enabled:         settings.TransferEnabled,
		FeeRate:         settings.TransferFeeRate,
		MinimumAmount:   settings.TransferMinAmount,
		MaximumAmount:   settings.TransferMaxAmount,
		DailyLimit:      settings.TransferDailyLimit,
		DailyCountLimit: settings.TransferDailyCountLimit,
		VIPFeeExempt:    settings.TransferVIPFeeExempt,
	}}, nil
}

// Service implements the direct-transfer slice without depending on BalanceTransferService.
type Service struct {
	repository    DirectTransferRepository
	settings      contract.SettingsReader
	users         directTransferUserReader
	subscriptions activeSubscriptionReader
	cache         balanceCacheInvalidator
}

// NewService constructs the direct-transfer service from narrow core adapters.
func NewService(
	repository DirectTransferRepository,
	settings contract.SettingsReader,
	users directTransferUserReader,
	subscriptions activeSubscriptionReader,
	cache balanceCacheInvalidator,
) *Service {
	return &Service{repository: repository, settings: settings, users: users, subscriptions: subscriptions, cache: cache}
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
	var numericID *int64
	if query != "" {
		if id, err := strconv.ParseInt(query, 10, 64); err == nil && id > 0 {
			numericID = &id
		} else if isASCIIDigits(query) {
			return contract.Recipient{}, ErrTransferReceiverQueryInvalid
		}
	}
	if numericID == nil && utf8.RuneCountInString(query) < 2 {
		return contract.Recipient{}, ErrTransferReceiverQueryInvalid
	}
	resolver, ok := s.users.(legacyRecipientResolver)
	if !ok {
		return contract.Recipient{}, ErrTransferReceiverNotFound
	}
	user, err := resolver.ResolveActiveTransferReceiver(ctx, query, numericID)
	if err != nil {
		return contract.Recipient{}, err
	}
	if user == nil || user.ID == requesterID || user.Status != service.StatusActive {
		return contract.Recipient{}, ErrTransferReceiverNotFound
	}
	return recipientFromUser(user), nil
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
	resolver, ok := s.users.(legacyRecipientResolver)
	if !ok {
		return []contract.RecipientCandidate{}, nil
	}
	users, err := resolver.SearchActiveTransferReceivers(ctx, query, requesterID, directTransferReceiverSearchLimit)
	if err != nil {
		return nil, err
	}
	results := make([]contract.RecipientCandidate, 0, min(len(users), directTransferReceiverSearchLimit))
	for _, user := range users {
		if user == nil || user.ID == requesterID || user.Status != service.StatusActive {
			continue
		}
		username := strings.TrimSpace(user.Username)
		email := maskRecipientEmail(user.Email)
		display := username
		if display == "" {
			display = email
		}
		results = append(results, contract.RecipientCandidate{AccountID: user.ID, DisplayName: display, Username: username, Email: email})
		if len(results) == directTransferReceiverSearchLimit {
			break
		}
	}
	return results, nil
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
	if s.users == nil {
		return contract.Recipient{}, ErrTransferReceiverNotFound
	}
	user, err := s.users.GetByID(ctx, accountID)
	if err != nil || user == nil || user.Status != service.StatusActive {
		return contract.Recipient{}, ErrTransferReceiverNotFound
	}
	return recipientFromUser(user), nil
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
	items, err := s.subscriptions.ListActiveUserSubscriptions(ctx, senderID)
	if err == nil && len(items) > 0 {
		return 0
	}
	return settings.FeeRate
}

func (s *Service) invalidateBalances(ctx context.Context, senderID, receiverID int64) {
	if s.cache == nil {
		return
	}
	_ = s.cache.InvalidateUserBalance(ctx, senderID)
	if receiverID != senderID {
		_ = s.cache.InvalidateUserBalance(ctx, receiverID)
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

func recipientFromUser(user *service.User) contract.Recipient {
	display := strings.TrimSpace(user.Username)
	if display == "" {
		display = maskRecipientEmail(user.Email)
	}
	if display == "" {
		display = service.UserDisplayName("", "", user.ID)
	}
	return contract.Recipient{Account: contract.Account{ID: user.ID, Role: user.Role, Status: user.Status, Balance: user.Balance, FrozenBalance: user.FrozenBalance}, DisplayName: display}
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
