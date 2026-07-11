//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestLeaderboardGetCheckinCountsUsesOneBatchQuery(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectQuery(`SELECT user_id, COUNT\(\*\) FROM checkins WHERE user_id IN \(\$1,\$2,\$3\) GROUP BY user_id`).
		WithArgs(int64(7), int64(11), int64(19)).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "count"}).AddRow(7, 4).AddRow(19, 2))

	service := &LeaderboardService{db: db}
	counts, err := service.getCheckinCounts(context.Background(), []int64{7, 11, 19})

	require.NoError(t, err)
	require.Equal(t, map[int64]int{7: 4, 19: 2}, counts)
	require.Zero(t, counts[11])
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLeaderboardGetCheckinCountsSkipsQueryForEmptyPage(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	service := &LeaderboardService{db: db}
	counts, err := service.getCheckinCounts(context.Background(), nil)

	require.NoError(t, err)
	require.Empty(t, counts)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMaskLeaderboardUsername(t *testing.T) {
	tests := []struct {
		name     string
		username string
		email    string
		want     string
	}{
		{name: "username", username: "alice", email: "ignored@example.com", want: "a***e"},
		{name: "unicode username", username: "张小明", want: "张***明"},
		{name: "two runes", username: "阿明", want: "阿*"},
		{name: "email local part", email: "person@example.com", want: "p***n"},
		{name: "single character", username: "x", want: "*"},
		{name: "anonymous fallback", want: "user"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, maskUsername(test.username, test.email))
		})
	}
}
