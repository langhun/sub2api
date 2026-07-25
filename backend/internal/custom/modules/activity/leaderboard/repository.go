package leaderboard

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/predicate"
	dbuser "github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/contract"
	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
)

// Repository owns the activity leaderboard's existing Ent and SQL read model.
// It preserves the legacy table layout without adding schema or migration work.
type Repository struct {
	entClient *dbent.Client
	db        *sql.DB
}

func NewRepository(client *dbent.Client, db *sql.DB) *Repository {
	return &Repository{entClient: client, db: db}
}

var (
	_ contract.BalanceLeaderboardReader     = (*Repository)(nil)
	_ contract.ConsumptionLeaderboardReader = (*Repository)(nil)
	_ contract.CheckinLeaderboardReader     = (*Repository)(nil)
	_ contract.TransferLeaderboardReader    = (*Repository)(nil)
)

func (r *Repository) ListBalanceLeaderboard(ctx context.Context, query contract.LeaderboardQuery) (contract.LeaderboardPage, error) {
	client, err := r.client(ctx)
	if err != nil {
		return contract.LeaderboardPage{}, err
	}
	offset := (query.Page - 1) * query.PageSize

	predicates := []predicate.User{
		dbuser.DeletedAtIsNil(),
		dbuser.StatusEQ(domain.StatusActive),
		dbuser.BalanceGT(0),
	}
	if !query.IncludeAdmin {
		predicates = append(predicates, dbuser.RoleNEQ("admin"))
	}
	total, err := client.User.Query().Where(predicates...).Count(ctx)
	if err != nil {
		return contract.LeaderboardPage{}, fmt.Errorf("count users: %w", err)
	}

	users, err := client.User.Query().Where(predicates...).
		Order(dbent.Desc(dbuser.FieldBalance), dbent.Asc(dbuser.FieldID)).
		Offset(offset).Limit(query.PageSize).All(ctx)
	if err != nil {
		return contract.LeaderboardPage{}, fmt.Errorf("query users: %w", err)
	}

	userIDs := make([]int64, 0, len(users))
	for _, user := range users {
		userIDs = append(userIDs, user.ID)
	}
	checkinCounts, err := r.getCheckinCounts(ctx, userIDs)
	if err != nil {
		return contract.LeaderboardPage{}, fmt.Errorf("query checkin counts: %w", err)
	}

	entries := make([]contract.LeaderboardEntry, 0, len(users))
	for index, user := range users {
		entries = append(entries, contract.LeaderboardEntry{
			Rank:     offset + index + 1,
			Username: leaderboardUserDisplay(user.Username, user.Email),
			Value:    math.Round(user.Balance*100) / 100,
			ExtraInt: checkinCounts[user.ID],
		})
	}
	return contract.LeaderboardPage{Entries: entries, Total: int64(total)}, nil
}

func (r *Repository) ListConsumptionLeaderboard(ctx context.Context, query contract.LeaderboardQuery) (contract.LeaderboardPage, error) {
	db, err := r.database()
	if err != nil {
		return contract.LeaderboardPage{}, err
	}
	today := timezone.Today()
	var startTime time.Time
	switch query.Period {
	case contract.LeaderboardPeriodDaily:
		startTime = today
	case contract.LeaderboardPeriodWeekly:
		startTime = today.AddDate(0, 0, -7)
	case contract.LeaderboardPeriodMonthly:
		startTime = today.AddDate(0, 0, -30)
	default:
		startTime = today
	}

	offset := (query.Page - 1) * query.PageSize
	roleClause := " AND u.role != 'admin'"
	if query.IncludeAdmin {
		roleClause = ""
	}
	countQuery := `
		SELECT COUNT(*) FROM (
			SELECT ul.user_id
			FROM usage_logs ul
			INNER JOIN users u ON ul.user_id = u.id AND u.deleted_at IS NULL
			WHERE ul.created_at >= $1 AND u.status = 'active'` + roleClause + `
			GROUP BY ul.user_id
			HAVING SUM(ul.actual_cost) > 0
		) sub
	`
	var total int64
	if err := db.QueryRowContext(ctx, countQuery, startTime).Scan(&total); err != nil {
		return contract.LeaderboardPage{}, fmt.Errorf("count consumption: %w", err)
	}

	dataQuery := `
		SELECT u.username, u.email, COALESCE(SUM(ul.actual_cost), 0) as total_cost, COUNT(*) as request_count
		FROM usage_logs ul
		INNER JOIN users u ON ul.user_id = u.id AND u.deleted_at IS NULL
		WHERE ul.created_at >= $1 AND u.status = 'active'` + roleClause + `
		GROUP BY ul.user_id, u.username, u.email
		HAVING SUM(ul.actual_cost) > 0
		ORDER BY total_cost DESC, ul.user_id ASC
		LIMIT $2 OFFSET $3
	`
	rows, err := db.QueryContext(ctx, dataQuery, startTime, query.PageSize, offset)
	if err != nil {
		return contract.LeaderboardPage{}, fmt.Errorf("query consumption: %w", err)
	}
	defer rows.Close()

	entries := make([]contract.LeaderboardEntry, 0)
	rank := offset
	for rows.Next() {
		rank++
		var username, email string
		var totalCost float64
		var requestCount int
		if err := rows.Scan(&username, &email, &totalCost, &requestCount); err != nil {
			return contract.LeaderboardPage{}, fmt.Errorf("scan consumption row: %w", err)
		}
		entries = append(entries, contract.LeaderboardEntry{
			Rank:     rank,
			Username: leaderboardUserDisplay(username, email),
			Value:    math.Round(totalCost*100) / 100,
			ExtraInt: requestCount,
		})
	}
	if err := rows.Err(); err != nil {
		return contract.LeaderboardPage{}, fmt.Errorf("iterate consumption rows: %w", err)
	}
	return contract.LeaderboardPage{Entries: entries, Total: total}, nil
}

func (r *Repository) ListCheckinLeaderboard(ctx context.Context, query contract.LeaderboardQuery) (contract.LeaderboardPage, error) {
	db, err := r.database()
	if err != nil {
		return contract.LeaderboardPage{}, err
	}
	today := timezone.Today()
	yesterday := today.AddDate(0, 0, -1)
	offset := (query.Page - 1) * query.PageSize

	roleClause := " AND u.role != 'admin'"
	if query.IncludeAdmin {
		roleClause = ""
	}
	countQuery := `
		SELECT COUNT(*) FROM (
			SELECT c.user_id
			FROM checkins c
			INNER JOIN (
				SELECT user_id, MAX(checkin_date) as max_date
				FROM checkins
				GROUP BY user_id
			) latest ON c.user_id = latest.user_id AND c.checkin_date = latest.max_date
			INNER JOIN users u ON c.user_id = u.id AND u.deleted_at IS NULL
			WHERE c.checkin_date >= $1 AND u.status = 'active'` + roleClause + `
		) sub
	`
	var total int64
	if err := db.QueryRowContext(ctx, countQuery, yesterday).Scan(&total); err != nil {
		return contract.LeaderboardPage{}, fmt.Errorf("count checkin: %w", err)
	}

	dataQuery := `
		SELECT u.username, u.email, c.streak_days, c.reward_amount,
			(SELECT COUNT(*) FROM checkins WHERE user_id = c.user_id) as total_checkins,
			(SELECT MAX(checkin_date) FROM checkins WHERE user_id = c.user_id) as last_date
		FROM checkins c
		INNER JOIN (
			SELECT user_id, MAX(checkin_date) as max_date
			FROM checkins
			GROUP BY user_id
		) latest ON c.user_id = latest.user_id AND c.checkin_date = latest.max_date
		INNER JOIN users u ON c.user_id = u.id AND u.deleted_at IS NULL
		WHERE c.checkin_date >= $1 AND u.status = 'active'` + roleClause + `
		ORDER BY c.streak_days DESC, c.checkin_date DESC, c.user_id ASC
		LIMIT $2 OFFSET $3
	`
	rows, err := db.QueryContext(ctx, dataQuery, yesterday, query.PageSize, offset)
	if err != nil {
		return contract.LeaderboardPage{}, fmt.Errorf("query checkin: %w", err)
	}
	defer rows.Close()

	entries := make([]contract.LeaderboardEntry, 0)
	rank := offset
	for rows.Next() {
		rank++
		var username, email string
		var streakDays int
		var rewardAmount float64
		var totalCheckins int
		var lastDate time.Time
		if err := rows.Scan(&username, &email, &streakDays, &rewardAmount, &totalCheckins, &lastDate); err != nil {
			return contract.LeaderboardPage{}, fmt.Errorf("scan checkin row: %w", err)
		}
		entries = append(entries, contract.LeaderboardEntry{
			Rank:       rank,
			Username:   leaderboardUserDisplay(username, email),
			Value:      float64(streakDays),
			ExtraInt:   totalCheckins,
			ExtraFloat: math.Round(rewardAmount*100) / 100,
			ExtraDate:  lastDate.Format("2006-01-02"),
		})
	}
	if err := rows.Err(); err != nil {
		return contract.LeaderboardPage{}, fmt.Errorf("iterate checkin rows: %w", err)
	}
	return contract.LeaderboardPage{Entries: entries, Total: total}, nil
}

func (r *Repository) ListTransferLeaderboard(ctx context.Context, query contract.LeaderboardQuery) (contract.LeaderboardPage, error) {
	db, err := r.database()
	if err != nil {
		return contract.LeaderboardPage{}, err
	}
	now := time.Now()
	var start time.Time
	switch query.Period {
	case contract.LeaderboardPeriodWeekly:
		start = now.AddDate(0, 0, -7)
	case contract.LeaderboardPeriodMonthly:
		start = now.AddDate(0, -1, 0)
	default:
		start = now.AddDate(0, 0, -1)
	}
	roleClause := " AND u.role != 'admin'"
	if query.IncludeAdmin {
		roleClause = ""
	}
	var total int64
	countQuery := `SELECT COUNT(*) FROM (
		SELECT bt.sender_id FROM balance_transfers bt
		JOIN users u ON u.id = bt.sender_id AND u.deleted_at IS NULL
		WHERE bt.status = 'completed' AND bt.transfer_type = 'direct'
		  AND bt.created_at >= $1 AND bt.created_at < $2 AND u.status = 'active'` + roleClause + `
		GROUP BY bt.sender_id
	) ranked`
	if err := db.QueryRowContext(ctx, countQuery, start, now).Scan(&total); err != nil {
		return contract.LeaderboardPage{}, fmt.Errorf("count transfer leaderboard: %w", err)
	}
	offset := (query.Page - 1) * query.PageSize
	dataQuery := `SELECT u.username, u.email, SUM(bt.amount), COUNT(*)
		FROM balance_transfers bt
		JOIN users u ON u.id = bt.sender_id AND u.deleted_at IS NULL
		WHERE bt.status = 'completed' AND bt.transfer_type = 'direct'
		  AND bt.created_at >= $1 AND bt.created_at < $2 AND u.status = 'active'` + roleClause + `
		GROUP BY bt.sender_id, u.username, u.email
		ORDER BY SUM(bt.amount) DESC, bt.sender_id ASC LIMIT $3 OFFSET $4`
	rows, err := db.QueryContext(ctx, dataQuery, start, now, query.PageSize, offset)
	if err != nil {
		return contract.LeaderboardPage{}, fmt.Errorf("query transfer leaderboard: %w", err)
	}
	defer rows.Close()

	entries := make([]contract.LeaderboardEntry, 0, query.PageSize)
	for rows.Next() {
		var username, email string
		var amount float64
		var count int
		if err := rows.Scan(&username, &email, &amount, &count); err != nil {
			return contract.LeaderboardPage{}, fmt.Errorf("scan transfer leaderboard: %w", err)
		}
		entries = append(entries, contract.LeaderboardEntry{
			Rank: offset + len(entries) + 1, Username: leaderboardUserDisplay(username, email),
			Value: math.Round(amount*1e8) / 1e8, ExtraInt: count,
		})
	}
	if err := rows.Err(); err != nil {
		return contract.LeaderboardPage{}, err
	}
	return contract.LeaderboardPage{Entries: entries, Total: total}, nil
}

func (r *Repository) client(ctx context.Context) (*dbent.Client, error) {
	if r == nil || r.entClient == nil {
		return nil, fmt.Errorf("activity leaderboard repository is unavailable")
	}
	if tx := dbent.TxFromContext(ctx); tx != nil {
		return tx.Client(), nil
	}
	return r.entClient, nil
}

func (r *Repository) database() (*sql.DB, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("activity leaderboard repository is unavailable")
	}
	return r.db, nil
}

func (r *Repository) getCheckinCounts(ctx context.Context, userIDs []int64) (map[int64]int, error) {
	counts := make(map[int64]int, len(userIDs))
	if len(userIDs) == 0 {
		return counts, nil
	}
	db, err := r.database()
	if err != nil {
		return nil, err
	}
	placeholders := make([]string, len(userIDs))
	args := make([]any, len(userIDs))
	for index, userID := range userIDs {
		placeholders[index] = fmt.Sprintf("$%d", index+1)
		args[index] = userID
	}
	query := `SELECT user_id, COUNT(*) FROM checkins WHERE user_id IN (` + strings.Join(placeholders, ",") + `) GROUP BY user_id`
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var userID int64
		var count int
		if err := rows.Scan(&userID, &count); err != nil {
			return nil, err
		}
		counts[userID] = count
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return counts, nil
}

func leaderboardUserDisplay(username, email string) string {
	if value := strings.TrimSpace(username); value != "" {
		return value
	}
	if value := strings.TrimSpace(email); value != "" {
		return maskEmail(value)
	}
	return "\u7528\u6237"
}

func maskEmail(email string) string {
	if len(email) < 3 {
		return "***"
	}
	atIndex := -1
	for index, character := range email {
		if character == '@' {
			atIndex = index
			break
		}
	}
	if atIndex == -1 || atIndex < 1 {
		return email[:1] + "***"
	}
	localPart := email[:atIndex]
	domain := email[atIndex:]
	if len(localPart) <= 2 {
		return localPart[:1] + "***" + domain
	}
	return localPart[:1] + "***" + localPart[len(localPart)-1:] + domain
}
