// Package supporttickets owns the custom support ticket HTTP surface.
package supporttickets

import "time"

const ModuleID = "support_tickets"

const (
	MaxMessagesPerTicket    = 100
	MaxActiveTicketsPerUser = 20
)

const (
	StatusOpen        = "open"
	StatusWaitingUser = "waiting_user"
	StatusResolved    = "resolved"
	StatusClosed      = "closed"
)

const (
	PriorityLow    = "low"
	PriorityNormal = "normal"
	PriorityHigh   = "high"
	PriorityUrgent = "urgent"
)

type Ticket struct {
	ID             int64      `json:"-"`
	PublicID       string     `json:"public_id"`
	UserID         int64      `json:"user_id"`
	UserName       string     `json:"user_name,omitempty"`
	Category       string     `json:"category"`
	Subject        string     `json:"subject"`
	Status         string     `json:"status"`
	Priority       string     `json:"priority"`
	LastReplyBy    string     `json:"last_reply_by"`
	LastActivityAt time.Time  `json:"last_activity_at"`
	ResolvedAt     *time.Time `json:"resolved_at,omitempty"`
	ClosedAt       *time.Time `json:"closed_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type Message struct {
	ID           int64     `json:"id"`
	AuthorType   string    `json:"author_type"`
	AuthorUserID int64     `json:"-"`
	MessageType  string    `json:"message_type"`
	Body         string    `json:"body"`
	CreatedAt    time.Time `json:"created_at"`
}

type TicketDetail struct {
	Ticket
	Messages []Message `json:"messages"`
}

type TicketEvent struct {
	ID          int64     `json:"id"`
	ActorType   string    `json:"actor_type"`
	ActorUserID int64     `json:"actor_user_id"`
	EventType   string    `json:"event_type"`
	OldValue    string    `json:"old_value,omitempty"`
	NewValue    string    `json:"new_value,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type CreateInput struct {
	UserID                  int64
	Category, Subject, Body string
}
type AdminCreateInput struct {
	AdminID, UserID         int64
	Category, Subject, Body string
}
type ReplyInput struct {
	UserID                    int64
	PublicID, Body, ActorType string
	Internal                  bool
}
type UpdateInput struct {
	UserID                     int64
	PublicID, Status, Priority string
}

type ListFilter struct {
	Page, PageSize   int
	Status, Priority string
	UserID           *int64
}
