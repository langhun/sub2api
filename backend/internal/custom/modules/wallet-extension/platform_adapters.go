package walletextension

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/predicate"
	dbuser "github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/ent/usersubscription"
	"github.com/Wei-Shaw/sub2api/internal/custom/modules/wallet-extension/contract"
)

// NewEntAccountReader projects only wallet account fields from Ent.
func NewEntAccountReader(client *dbent.Client) contract.AccountReader {
	return entAccountReader{client: client}
}

type entAccountReader struct{ client *dbent.Client }

func (r entAccountReader) GetAccount(ctx context.Context, accountID int64) (contract.Account, error) {
	if r.client == nil || accountID <= 0 {
		return contract.Account{}, fmt.Errorf("invalid wallet account lookup")
	}
	item, err := walletEntClient(ctx, r.client).User.Query().Where(dbuser.IDEQ(accountID)).Only(ctx)
	if err != nil {
		return contract.Account{}, err
	}
	return walletAccountFromEnt(item), nil
}

// NewEntRecipientResolver owns wallet recipient lookup and returns only
// response-safe wallet DTOs.
func NewEntRecipientResolver(client *dbent.Client) contract.RecipientResolver {
	return entRecipientResolver{client: client}
}

type entRecipientResolver struct{ client *dbent.Client }

func (r entRecipientResolver) ResolveDirectTransferRecipient(ctx context.Context, requesterID int64, query string) (contract.Recipient, error) {
	if r.client == nil {
		return contract.Recipient{}, fmt.Errorf("direct transfer recipient resolver is unavailable")
	}
	predicates := []predicate.User{dbuser.StatusEQ(accountStatusActive)}
	if numericID, err := strconv.ParseInt(query, 10, 64); err == nil && numericID > 0 {
		predicates = append(predicates, dbuser.IDEQ(numericID))
	} else {
		predicates = append(predicates, dbuser.Or(
			dbuser.EmailEqualFold(query),
			dbuser.UsernameEqualFold(query),
		))
	}
	items, err := walletEntClient(ctx, r.client).User.Query().Where(predicates...).Order(dbent.Asc(dbuser.FieldID)).Limit(2).All(ctx)
	if err != nil {
		return contract.Recipient{}, fmt.Errorf("resolve active transfer receiver: %w", err)
	}
	if len(items) == 0 || items[0].ID == requesterID {
		return contract.Recipient{}, ErrTransferReceiverNotFound
	}
	if len(items) > 1 {
		return contract.Recipient{}, ErrTransferReceiverAmbiguous
	}
	return recipientFromAccount(walletAccountFromEnt(items[0])), nil
}

func (r entRecipientResolver) SearchDirectTransferRecipients(ctx context.Context, requesterID int64, query string, limit int) ([]contract.RecipientCandidate, error) {
	if limit <= 0 {
		return []contract.RecipientCandidate{}, nil
	}
	if r.client == nil {
		return nil, fmt.Errorf("direct transfer recipient resolver is unavailable")
	}
	var identityPredicate predicate.User
	if numericID, err := strconv.ParseInt(query, 10, 64); err == nil && numericID > 0 {
		identityPredicate = dbuser.IDEQ(numericID)
	} else {
		identityPredicate = dbuser.Or(
			dbuser.EmailContainsFold(query),
			dbuser.UsernameContainsFold(query),
		)
	}
	items, err := walletEntClient(ctx, r.client).User.Query().Where(
		dbuser.StatusEQ(accountStatusActive),
		dbuser.IDNEQ(requesterID),
		identityPredicate,
	).Order(dbent.Asc(dbuser.FieldID)).Limit(limit).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("search active transfer receivers: %w", err)
	}
	results := make([]contract.RecipientCandidate, 0, len(items))
	for _, item := range items {
		account := walletAccountFromEnt(item)
		username := strings.TrimSpace(account.Username)
		email := maskRecipientEmail(account.Email)
		display := username
		if display == "" {
			display = email
		}
		results = append(results, contract.RecipientCandidate{
			AccountID: account.ID, DisplayName: display, Username: username, Email: email,
		})
	}
	return results, nil
}

// NewEntActiveSubscriptionReader preserves the original active-subscription
// predicate without exposing subscription records to wallet behavior.
func NewEntActiveSubscriptionReader(client *dbent.Client) contract.ActiveSubscriptionReader {
	return entActiveSubscriptionReader{client: client}
}

type entActiveSubscriptionReader struct{ client *dbent.Client }

func (r entActiveSubscriptionReader) HasActiveSubscription(ctx context.Context, accountID int64) (bool, error) {
	if r.client == nil || accountID <= 0 {
		return false, fmt.Errorf("invalid wallet subscription lookup")
	}
	return walletEntClient(ctx, r.client).UserSubscription.Query().Where(
		usersubscription.UserIDEQ(accountID),
		usersubscription.StatusEQ(accountStatusActive),
		usersubscription.ExpiresAtGT(time.Now()),
		usersubscription.DeletedAtIsNil(),
	).Exist(ctx)
}

// NewEntBalanceWriter uses the wallet balance port without passing a core user
// repository into the module.
func NewEntBalanceWriter(client *dbent.Client) contract.BalanceWriter {
	return entBalanceWriter{client: client}
}

type entBalanceWriter struct{ client *dbent.Client }

func (w entBalanceWriter) Credit(ctx context.Context, operation contract.BalanceOperation) error {
	if w.client == nil || operation.AccountID <= 0 || operation.Amount <= 0 {
		return fmt.Errorf("invalid wallet balance credit")
	}
	updated, err := walletEntClient(ctx, w.client).User.Update().Where(dbuser.IDEQ(operation.AccountID)).AddBalance(operation.Amount).Save(ctx)
	if err != nil {
		return err
	}
	if updated != 1 {
		return fmt.Errorf("wallet account %d not found", operation.AccountID)
	}
	return nil
}

func (w entBalanceWriter) DebitIfSufficient(ctx context.Context, operation contract.BalanceOperation) (bool, error) {
	if w.client == nil || operation.AccountID <= 0 || operation.Amount <= 0 {
		return false, fmt.Errorf("invalid wallet balance debit")
	}
	result, err := walletEntClient(ctx, w.client).ExecContext(ctx, `
		UPDATE users SET balance = balance - $1, updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL AND balance >= $1`, operation.Amount, operation.AccountID)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	return affected == 1, err
}

// BalanceCacheSource is the only cache capability wallet requires.
type BalanceCacheSource interface {
	InvalidateUserBalance(ctx context.Context, userID int64) error
}

func NewBalanceCacheInvalidator(source BalanceCacheSource) contract.BalanceCacheInvalidator {
	return balanceCacheInvalidator{source: source}
}

type balanceCacheInvalidator struct{ source BalanceCacheSource }

func (i balanceCacheInvalidator) InvalidateBalance(ctx context.Context, accountID int64) error {
	if i.source == nil {
		return nil
	}
	return i.source.InvalidateUserBalance(ctx, accountID)
}

func walletAccountFromEnt(item *dbent.User) contract.Account {
	if item == nil {
		return contract.Account{}
	}
	return contract.Account{
		ID: item.ID, Role: item.Role, Status: item.Status, Balance: item.Balance, FrozenBalance: item.FrozenBalance,
		Username: item.Username, Email: item.Email,
	}
}

func walletEntClient(ctx context.Context, fallback *dbent.Client) *dbent.Client {
	if tx := dbent.TxFromContext(ctx); tx != nil {
		return tx.Client()
	}
	return fallback
}

var (
	_ contract.AccountReader            = entAccountReader{}
	_ contract.RecipientResolver        = entRecipientResolver{}
	_ contract.ActiveSubscriptionReader = entActiveSubscriptionReader{}
	_ contract.BalanceWriter            = entBalanceWriter{}
	_ contract.BalanceCacheInvalidator  = balanceCacheInvalidator{}
)
