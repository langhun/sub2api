package supporttickets

import (
	"context"
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
	columns := []string{"id", "public_id", "user_id", "category", "subject", "status", "priority", "last_reply_by", "last_activity_at", "resolved_at", "closed_at", "created_at", "updated_at"}
	mock.ExpectQuery(`FROM custom_support_tickets WHERE public_id = \$1 AND user_id = \$2`).
		WithArgs("SUP-0123456789ABCDEF", int64(7)).
		WillReturnRows(sqlmock.NewRows(columns).AddRow(int64(1), "SUP-0123456789ABCDEF", int64(7), "api", "Help", StatusOpen, PriorityNormal, "user", now, nil, nil, now, now))
	mock.ExpectQuery(`FROM custom_support_ticket_messages WHERE ticket_id = \$1 AND message_type = 'reply'`).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "author_type", "author_user_id", "message_type", "body", "created_at"}).AddRow(int64(2), "user", int64(7), "reply", "message", now))

	ownerID := int64(7)
	detail, err := NewRepository(db).Detail(context.Background(), "SUP-0123456789ABCDEF", &ownerID, false)
	require.NoError(t, err)
	require.Len(t, detail.Messages, 1)
	require.Equal(t, "reply", detail.Messages[0].MessageType)
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
