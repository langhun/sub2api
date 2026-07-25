// Package contract defines the core capabilities consumed by wallet-extension.
package contract

import "context"

// Account is the minimum user state required by wallet operations.
type Account struct {
	ID            int64   `json:"id"`
	Role          string  `json:"role"`
	Status        string  `json:"status"`
	Balance       float64 `json:"balance"`
	FrozenBalance float64 `json:"frozen_balance"`
	Username      string  `json:"-"`
	Email         string  `json:"-"`
}

// Recipient is an account that may receive a direct transfer.
type Recipient struct {
	Account     Account `json:"-"`
	DisplayName string  `json:"receiver_display"`
}

// RecipientCandidate is a deliberately minimal searchable recipient result.
type RecipientCandidate struct {
	AccountID   int64  `json:"receiver_id"`
	DisplayName string `json:"receiver_display"`
	Username    string `json:"receiver_username"`
	Email       string `json:"receiver_email"`
}

// AccountReader resolves account state before a wallet operation.
type AccountReader interface {
	GetAccount(ctx context.Context, accountID int64) (Account, error)
}

// RecipientResolver resolves and searches active direct-transfer recipients.
type RecipientResolver interface {
	ResolveDirectTransferRecipient(ctx context.Context, requesterID int64, query string) (Recipient, error)
	SearchDirectTransferRecipients(ctx context.Context, requesterID int64, query string, limit int) ([]RecipientCandidate, error)
}

// ActiveSubscriptionReader answers the one subscription question needed for
// direct-transfer fee exemption without exposing subscription records.
type ActiveSubscriptionReader interface {
	HasActiveSubscription(ctx context.Context, accountID int64) (bool, error)
}
