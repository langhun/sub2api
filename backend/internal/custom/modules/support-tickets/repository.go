package supporttickets

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var (
	ErrNotFound          = errors.New("support ticket not found")
	ErrTicketInactive    = errors.New("support ticket must be reopened before replying")
	ErrInvalidTransition = errors.New("ticket status transition is not allowed")
)

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Create(ctx context.Context, ticket Ticket, body string) (*Ticket, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("support ticket database is unavailable")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin support ticket: %w", err)
	}
	defer tx.Rollback()
	row := tx.QueryRowContext(ctx, `INSERT INTO custom_support_tickets (public_id, user_id, category, subject, status, priority, last_reply_by, user_last_read_at, last_activity_at, created_at, updated_at) VALUES ($1,$2,$3,$4,'open','normal','user',NOW(),NOW(),NOW(),NOW()) RETURNING id, last_activity_at, created_at, updated_at`, ticket.PublicID, ticket.UserID, ticket.Category, ticket.Subject)
	if err := row.Scan(&ticket.ID, &ticket.LastActivityAt, &ticket.CreatedAt, &ticket.UpdatedAt); err != nil {
		return nil, fmt.Errorf("insert support ticket: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO custom_support_ticket_messages (ticket_id, author_type, author_user_id, message_type, body) VALUES ($1,'user',$2,'reply',$3)`, ticket.ID, ticket.UserID, body); err != nil {
		return nil, fmt.Errorf("insert support ticket message: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO custom_support_ticket_events (ticket_id, actor_type, actor_user_id, event_type, new_value) VALUES ($1,'user',$2,'created','open')`, ticket.ID, ticket.UserID); err != nil {
		return nil, fmt.Errorf("insert support ticket event: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit support ticket: %w", err)
	}
	ticket.Status = StatusOpen
	ticket.Priority = PriorityNormal
	ticket.LastReplyBy = "user"
	return &ticket, nil
}

func (r *Repository) List(ctx context.Context, filter ListFilter) ([]Ticket, int64, error) {
	if r == nil || r.db == nil {
		return nil, 0, errors.New("support ticket database is unavailable")
	}
	where := "WHERE 1=1"
	args := []any{}
	n := 0
	add := func(condition string, value any) {
		n++
		where += fmt.Sprintf(" AND %s = $%d", condition, n)
		args = append(args, value)
	}
	if filter.UserID != nil {
		add("user_id", *filter.UserID)
	}
	if filter.Status != "" {
		add("status", filter.Status)
	}
	if filter.Priority != "" {
		add("priority", filter.Priority)
	}
	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM custom_support_tickets "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count support tickets: %w", err)
	}
	n++
	limitArg := n
	args = append(args, filter.PageSize)
	n++
	offsetArg := n
	args = append(args, (filter.Page-1)*filter.PageSize)
	rows, err := r.db.QueryContext(ctx, `SELECT id, public_id, user_id, category, subject, status, priority, last_reply_by, last_activity_at, resolved_at, closed_at, created_at, updated_at FROM custom_support_tickets `+where+fmt.Sprintf(" ORDER BY last_activity_at DESC, id DESC LIMIT $%d OFFSET $%d", limitArg, offsetArg), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list support tickets: %w", err)
	}
	defer rows.Close()
	items := make([]Ticket, 0)
	for rows.Next() {
		var item Ticket
		if err := scanTicket(rows, &item); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate support tickets: %w", err)
	}
	return items, total, nil
}

func (r *Repository) Detail(ctx context.Context, publicID string, userID *int64, includeInternal bool) (*TicketDetail, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("support ticket database is unavailable")
	}
	args := []any{publicID}
	clause := "public_id = $1"
	if userID != nil {
		args = append(args, *userID)
		clause += " AND user_id = $2"
	}
	var result TicketDetail
	err := scanTicket(r.db.QueryRowContext(ctx, `SELECT id, public_id, user_id, category, subject, status, priority, last_reply_by, last_activity_at, resolved_at, closed_at, created_at, updated_at FROM custom_support_tickets WHERE `+clause, args...), &result.Ticket)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get support ticket: %w", err)
	}
	messageWhere := "ticket_id = $1"
	if !includeInternal {
		messageWhere += " AND message_type = 'reply'"
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, author_type, author_user_id, message_type, body, created_at FROM custom_support_ticket_messages WHERE `+messageWhere+` ORDER BY created_at ASC, id ASC`, result.ID)
	if err != nil {
		return nil, fmt.Errorf("list ticket messages: %w", err)
	}
	defer rows.Close()
	result.Messages = make([]Message, 0)
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.AuthorType, &m.AuthorUserID, &m.MessageType, &m.Body, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan ticket message: %w", err)
		}
		result.Messages = append(result.Messages, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate ticket messages: %w", err)
	}
	return &result, nil
}

func (r *Repository) AddReply(ctx context.Context, input ReplyInput) (*Ticket, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("support ticket database is unavailable")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin ticket reply: %w", err)
	}
	defer tx.Rollback()
	var ticket Ticket
	if err := scanTicket(tx.QueryRowContext(ctx, `SELECT id, public_id, user_id, category, subject, status, priority, last_reply_by, last_activity_at, resolved_at, closed_at, created_at, updated_at FROM custom_support_tickets WHERE public_id = $1 FOR UPDATE`, input.PublicID), &ticket); errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, fmt.Errorf("lock support ticket: %w", err)
	}
	if input.ActorType == "user" && ticket.UserID != input.UserID {
		return nil, ErrNotFound
	}
	if input.ActorType == "user" && !isAllowedUserReply(ticket.Status) {
		return nil, ErrTicketInactive
	}
	if input.ActorType == "admin" && !input.Internal && (ticket.Status == StatusResolved || ticket.Status == StatusClosed) {
		return nil, ErrTicketInactive
	}
	author := input.ActorType
	kind := "reply"
	if input.Internal {
		kind = "internal_note"
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO custom_support_ticket_messages (ticket_id, author_type, author_user_id, message_type, body) VALUES ($1,$2,$3,$4,$5)`, ticket.ID, author, input.UserID, kind, input.Body); err != nil {
		return nil, fmt.Errorf("insert ticket reply: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO custom_support_ticket_events (ticket_id, actor_type, actor_user_id, event_type, new_value) VALUES ($1,$2,$3,$4,$5)`, ticket.ID, author, input.UserID, kind+"_added", kind); err != nil {
		return nil, fmt.Errorf("insert ticket reply event: %w", err)
	}
	status := ticket.Status
	if author == "user" && status == StatusWaitingUser {
		status = StatusOpen
	}
	if author == "admin" && !input.Internal && status == StatusOpen {
		status = StatusWaitingUser
	}
	var resolvedAt, closedAt any = ticket.ResolvedAt, ticket.ClosedAt
	if status == StatusOpen || status == StatusWaitingUser {
		resolvedAt = nil
		closedAt = nil
	}
	if _, err := tx.ExecContext(ctx, `UPDATE custom_support_tickets SET status=$1,last_reply_by=$2::text,last_activity_at=NOW(),updated_at=NOW(),user_last_read_at=CASE WHEN $2::text='user' THEN NOW() ELSE user_last_read_at END,admin_last_read_at=CASE WHEN $2::text='admin' THEN NOW() ELSE admin_last_read_at END,resolved_at=$3,closed_at=$4 WHERE id=$5`, status, author, resolvedAt, closedAt, ticket.ID); err != nil {
		return nil, fmt.Errorf("update ticket reply: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit ticket reply: %w", err)
	}
	ticket.Status = status
	ticket.LastReplyBy = author
	ticket.LastActivityAt = time.Now().UTC()
	return &ticket, nil
}

func (r *Repository) Reopen(ctx context.Context, publicID string, userID int64) (*Ticket, error) {
	return r.update(ctx, publicID, userID, "user", "status", StatusOpen)
}
func (r *Repository) UpdateStatus(ctx context.Context, publicID string, adminID int64, status string) (*Ticket, error) {
	return r.update(ctx, publicID, adminID, "admin", "status", status)
}
func (r *Repository) UpdatePriority(ctx context.Context, publicID string, adminID int64, priority string) (*Ticket, error) {
	return r.update(ctx, publicID, adminID, "admin", "priority", priority)
}
func (r *Repository) update(ctx context.Context, publicID string, actorID int64, actorType, field, value string) (*Ticket, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("support ticket database is unavailable")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin ticket update: %w", err)
	}
	defer tx.Rollback()
	var t Ticket
	if err := scanTicket(tx.QueryRowContext(ctx, `SELECT id, public_id, user_id, category, subject, status, priority, last_reply_by, last_activity_at, resolved_at, closed_at, created_at, updated_at FROM custom_support_tickets WHERE public_id=$1 FOR UPDATE`, publicID), &t); errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}
	if actorType == "user" && t.UserID != actorID {
		return nil, ErrNotFound
	}
	if field == "status" {
		if actorType == "user" {
			if !isAllowedUserStatusTransition(t.Status, value) {
				return nil, ErrInvalidTransition
			}
		} else if !isAllowedAdminStatusTransition(t.Status, value) {
			return nil, ErrInvalidTransition
		}
	}
	old := t.Status
	if field == "priority" {
		old = t.Priority
	}
	query := "UPDATE custom_support_tickets SET " + field + "=$1, last_activity_at=NOW(), updated_at=NOW()"
	if field == "status" {
		query += statusTimestampAssignments(t.Status, value)
	}
	query += " WHERE id=$2"
	if _, err := tx.ExecContext(ctx, query, value, t.ID); err != nil {
		return nil, fmt.Errorf("update ticket %s: %w", field, err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO custom_support_ticket_events (ticket_id, actor_type, actor_user_id, event_type, old_value, new_value) VALUES ($1,$2,$3,$4,$5,$6)`, t.ID, actorType, actorID, field+"_changed", old, value); err != nil {
		return nil, fmt.Errorf("insert ticket event: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit ticket update: %w", err)
	}
	if field == "status" {
		t.Status = value
	} else {
		t.Priority = value
	}
	t.LastActivityAt = time.Now().UTC()
	return &t, nil
}

func isAllowedAdminStatusTransition(from, to string) bool {
	if from == to || from == StatusClosed {
		return false
	}
	switch from {
	case StatusOpen, StatusWaitingUser:
		return to == StatusOpen || to == StatusWaitingUser || to == StatusResolved || to == StatusClosed
	case StatusResolved:
		return to == StatusOpen || to == StatusClosed
	default:
		return false
	}
}

// A user may add a message only while the ticket is actively being handled.
// Resolved tickets must first take the explicit resolved -> open path.
func isAllowedUserReply(status string) bool {
	return status == StatusOpen || status == StatusWaitingUser
}

func isAllowedUserStatusTransition(from, to string) bool {
	return from == StatusResolved && to == StatusOpen
}

// Timestamp values are derived from the transition, rather than inferred from
// a previous row. This keeps terminal state metadata coherent after every
// allowed transition, including legacy rows with stale timestamp columns.
func statusTimestampAssignments(from, to string) string {
	switch to {
	case StatusResolved:
		return ", resolved_at=NOW(), closed_at=NULL"
	case StatusClosed:
		if from == StatusResolved {
			return ", closed_at=NOW()"
		}
		return ", resolved_at=NULL, closed_at=NOW()"
	default:
		return ", resolved_at=NULL, closed_at=NULL"
	}
}

type scanner interface{ Scan(...any) error }

func scanTicket(row scanner, t *Ticket) error {
	return row.Scan(&t.ID, &t.PublicID, &t.UserID, &t.Category, &t.Subject, &t.Status, &t.Priority, &t.LastReplyBy, &t.LastActivityAt, &t.ResolvedAt, &t.ClosedAt, &t.CreatedAt, &t.UpdatedAt)
}
