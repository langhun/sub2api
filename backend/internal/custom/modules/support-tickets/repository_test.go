package supporttickets

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestDetailScopesTicketToOwnerAndHidesInternalNotes(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	now := time.Now().UTC()
	columns := []string{"id", "public_id", "user_id", "category", "subject", "status", "priority", "last_reply_by", "last_activity_at", "resolved_at", "closed_at", "created_at", "updated_at", "username"}
	mock.ExpectQuery(`FROM custom_support_tickets t LEFT JOIN users u ON u.id = t.user_id WHERE t.public_id = \$1 AND t.user_id = \$2`).
		WithArgs("SUP-0123456789ABCDEF", int64(7)).
		WillReturnRows(sqlmock.NewRows(columns).AddRow(int64(1), "SUP-0123456789ABCDEF", int64(7), "api", "Help", StatusOpen, PriorityNormal, "user", now, nil, nil, now, now, "alice"))
	mock.ExpectQuery(`FROM custom_support_ticket_messages WHERE ticket_id = \$1 AND message_type = 'reply'`).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "author_type", "author_user_id", "message_type", "body", "created_at"}).AddRow(int64(2), "user", int64(7), "reply", "message", now))

	ownerID := int64(7)
	detail, err := NewRepository(db).Detail(context.Background(), "SUP-0123456789ABCDEF", &ownerID, false)
	require.NoError(t, err)
	require.Len(t, detail.Messages, 1)
	require.Equal(t, "reply", detail.Messages[0].MessageType)
	require.Equal(t, "alice", detail.UserName)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListIncludesRequesterUsername(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	now := time.Now().UTC()
	columns := []string{"id", "public_id", "user_id", "category", "subject", "status", "priority", "last_reply_by", "last_activity_at", "resolved_at", "closed_at", "created_at", "updated_at", "username"}
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM custom_support_tickets t WHERE 1=1 AND t.status = \$1`).
		WithArgs(StatusOpen).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`FROM custom_support_tickets t LEFT JOIN users u ON u.id = t.user_id WHERE 1=1 AND t.status = \$1 ORDER BY t.last_activity_at DESC, t.id DESC LIMIT \$2 OFFSET \$3`).
		WithArgs(StatusOpen, 20, 0).
		WillReturnRows(sqlmock.NewRows(columns).AddRow(int64(1), "SUP-0123456789ABCDEF", int64(7), "api", "Help", StatusOpen, PriorityNormal, "user", now, nil, nil, now, now, "alice"))

	items, total, err := NewRepository(db).List(context.Background(), ListFilter{Page: 1, PageSize: 20, Status: StatusOpen})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, items, 1)
	require.Equal(t, "alice", items[0].UserName)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAdminReplyUsesOneExplicitTextParameterType(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	now := time.Now().UTC()
	columns := []string{"id", "public_id", "user_id", "category", "subject", "status", "priority", "last_reply_by", "last_activity_at", "resolved_at", "closed_at", "created_at", "updated_at"}
	mock.ExpectBegin()
	mock.ExpectQuery(`FROM custom_support_tickets WHERE public_id = \$1 FOR UPDATE`).
		WithArgs("SUP-0123456789ABCDEF").
		WillReturnRows(sqlmock.NewRows(columns).AddRow(int64(1), "SUP-0123456789ABCDEF", int64(7), "api", "Help", StatusOpen, PriorityNormal, "user", now, nil, nil, now, now))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM custom_support_ticket_messages WHERE ticket_id = \$1`).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	mock.ExpectExec(`INSERT INTO custom_support_ticket_messages`).
		WithArgs(int64(1), "admin", int64(9), "reply", "response").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO custom_support_ticket_events`).
		WithArgs(int64(1), "admin", int64(9), "reply_added", "reply").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`last_reply_by=\$2::text.*CASE WHEN \$2::text='user'.*CASE WHEN \$2::text='admin'`).
		WithArgs(StatusWaitingUser, "admin", nil, nil, int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	ticket, err := NewRepository(db).AddReply(context.Background(), ReplyInput{
		PublicID:  "SUP-0123456789ABCDEF",
		UserID:    9,
		ActorType: "admin",
		Body:      "response",
	})
	require.NoError(t, err)
	require.Equal(t, StatusWaitingUser, ticket.Status)
	require.Equal(t, "admin", ticket.LastReplyBy)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateRejectsWhenActiveTicketLimitReached(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM custom_support_tickets WHERE user_id = \$1 AND status <> 'closed'`).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(MaxActiveTicketsPerUser))
	mock.ExpectRollback()

	_, err = NewRepository(db).Create(context.Background(), Ticket{PublicID: "SUP-0123456789ABCDEF", UserID: 7, Category: "api", Subject: "Help"}, "message")
	require.True(t, errors.Is(err, ErrTicketLimit))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateAdminCreatesUnreadOutboundTicketForActiveUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	now := time.Now().UTC()
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id FROM users WHERE id = \$1 AND status = 'active' AND deleted_at IS NULL FOR UPDATE`).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(7)))
	mock.ExpectQuery(`INSERT INTO custom_support_tickets .*'waiting_user'.*'admin'.*RETURNING id, last_activity_at, created_at, updated_at`).
		WithArgs("SUP-0123456789ABCDEF", int64(7), "account", "Account notice").
		WillReturnRows(sqlmock.NewRows([]string{"id", "last_activity_at", "created_at", "updated_at"}).AddRow(int64(1), now, now, now))
	mock.ExpectExec(`INSERT INTO custom_support_ticket_messages`).
		WithArgs(int64(1), int64(9), "Please review this ticket.").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO custom_support_ticket_events`).
		WithArgs(int64(1), int64(9)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	ticket, err := NewRepository(db).CreateAdmin(context.Background(), Ticket{PublicID: "SUP-0123456789ABCDEF", UserID: 7, Category: "account", Subject: "Account notice"}, 9, "Please review this ticket.")
	require.NoError(t, err)
	require.Equal(t, StatusWaitingUser, ticket.Status)
	require.Equal(t, PriorityNormal, ticket.Priority)
	require.Equal(t, "admin", ticket.LastReplyBy)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateAdminRejectsInactiveOrMissingTargetUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id FROM users WHERE id = \$1 AND status = 'active' AND deleted_at IS NULL FOR UPDATE`).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectRollback()

	_, err = NewRepository(db).CreateAdmin(context.Background(), Ticket{PublicID: "SUP-0123456789ABCDEF", UserID: 7, Category: "account", Subject: "Account notice"}, 9, "Please review this ticket.")
	require.ErrorIs(t, err, ErrTargetUserUnavailable)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestReplyRejectsWhenMessageLimitReached(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	now := time.Now().UTC()
	columns := []string{"id", "public_id", "user_id", "category", "subject", "status", "priority", "last_reply_by", "last_activity_at", "resolved_at", "closed_at", "created_at", "updated_at"}
	mock.ExpectBegin()
	mock.ExpectQuery(`FROM custom_support_tickets WHERE public_id = \$1 FOR UPDATE`).
		WithArgs("SUP-0123456789ABCDEF").
		WillReturnRows(sqlmock.NewRows(columns).AddRow(int64(1), "SUP-0123456789ABCDEF", int64(7), "api", "Help", StatusOpen, PriorityNormal, "user", now, nil, nil, now, now))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM custom_support_ticket_messages WHERE ticket_id = \$1`).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(MaxMessagesPerTicket))
	mock.ExpectRollback()

	_, err = NewRepository(db).AddReply(context.Background(), ReplyInput{PublicID: "SUP-0123456789ABCDEF", UserID: 7, ActorType: "user", Body: "response"})
	require.True(t, errors.Is(err, ErrTicketMessageLimit))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInternalNotePreservesUserVisibleTicketMetadata(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	now := time.Now().UTC()
	columns := []string{"id", "public_id", "user_id", "category", "subject", "status", "priority", "last_reply_by", "last_activity_at", "resolved_at", "closed_at", "created_at", "updated_at"}
	mock.ExpectBegin()
	mock.ExpectQuery(`FROM custom_support_tickets WHERE public_id = \$1 FOR UPDATE`).
		WithArgs("SUP-0123456789ABCDEF").
		WillReturnRows(sqlmock.NewRows(columns).AddRow(int64(1), "SUP-0123456789ABCDEF", int64(7), "api", "Help", StatusOpen, PriorityNormal, "user", now, nil, nil, now, now))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM custom_support_ticket_messages WHERE ticket_id = \$1`).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectExec(`INSERT INTO custom_support_ticket_messages`).
		WithArgs(int64(1), "admin", int64(9), "internal_note", "private note").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO custom_support_ticket_events`).
		WithArgs(int64(1), "admin", int64(9), "internal_note_added", "internal_note").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	ticket, err := NewRepository(db).AddReply(context.Background(), ReplyInput{PublicID: "SUP-0123456789ABCDEF", UserID: 9, ActorType: "admin", Body: "private note", Internal: true})
	require.NoError(t, err)
	require.Equal(t, StatusOpen, ticket.Status)
	require.Equal(t, "user", ticket.LastReplyBy)
	require.Equal(t, now, ticket.LastActivityAt)
	require.NoError(t, mock.ExpectationsWereMet())
}
