package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type userDisplayQuerier interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func queryUserDisplayNames(ctx context.Context, querier userDisplayQuerier, userIDs []int64) (map[int64]string, error) {
	unique := make(map[int64]struct{}, len(userIDs))
	for _, userID := range userIDs {
		if userID > 0 {
			unique[userID] = struct{}{}
		}
	}
	displays := make(map[int64]string, len(unique))
	if len(unique) == 0 {
		return displays, nil
	}

	args := make([]any, 0, len(unique))
	placeholders := make([]string, 0, len(unique))
	for userID := range unique {
		args = append(args, userID)
		placeholders = append(placeholders, fmt.Sprintf("$%d", len(args)))
	}
	rows, err := querier.QueryContext(ctx, `SELECT id, COALESCE(username, ''), COALESCE(email, '') FROM users WHERE id IN (`+strings.Join(placeholders, ",")+`)`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var userID int64
		var username, email string
		if err := rows.Scan(&userID, &username, &email); err != nil {
			return nil, err
		}
		displays[userID] = service.MaskedUserDisplayName(username, email, userID)
	}
	return displays, rows.Err()
}
