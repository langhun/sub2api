package contract

import (
	"context"
	"time"
)

// AuditEntry records a completed activity adjustment without exposing a core persistence model.
type AuditEntry struct {
	UserID         int64
	Type           string
	Amount         float64
	ReferenceID    string
	IdempotencyKey string
	OccurredAt     time.Time
	CodeType       string
	Notes          string
	GroupID        *int64
	ValidityDays   int
}

// AuditWriter records activity adjustments within the caller's transaction context.
type AuditWriter interface {
	WriteActivityAudit(ctx context.Context, entry AuditEntry) error
}

// InvitationCodeRequest describes an invitation reward generated for a blind-box delivery.
type InvitationCodeRequest struct {
	UserID         int64
	Code           string
	IdempotencyKey string
	ExpiresAt      *time.Time
}

// InvitationCodeIssuer creates an activity invitation reward exactly once per idempotency key.
type InvitationCodeIssuer interface {
	IssueInvitationCode(ctx context.Context, request InvitationCodeRequest) (string, error)
}

// SubscriptionGrant describes a subscription reward selected by a blind box.
type SubscriptionGrant struct {
	UserID         int64
	SubscriptionID int64
	Days           int
	IdempotencyKey string
	Note           string
}

// SubscriptionGranter grants or extends a subscription exactly once per idempotency key.
type SubscriptionGranter interface {
	GrantOrExtendSubscription(ctx context.Context, grant SubscriptionGrant) error
}

// Notification is an optional user-facing activity event.
type Notification struct {
	UserID    int64
	Type      string
	Title     string
	Content   string
	Reference string
}

// NotificationPublisher emits an activity notification after the underlying transaction commits.
type NotificationPublisher interface {
	PublishActivityNotification(ctx context.Context, notification Notification) error
}
