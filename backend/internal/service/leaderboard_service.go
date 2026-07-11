package service

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
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
)

type LeaderboardEntry struct {
	Rank       int     `json:"rank"`
	Username   string  `json:"username"`
	Value      float64 `json:"value"`
	ExtraInt   int     `json:"extra_int,omitempty"`
	ExtraInt2  int     `json:"extra_int2,omitempty"`
	ExtraFloat float64 `json:"extra_float,omitempty"`
	ExtraDate  string  `json:"extra_date,omitempty"`
}

type LeaderboardResult struct {
	Entries []LeaderboardEntry `json:"items"`
	Total   int64              `json:"total"`
}

type LeaderboardService struct {
	entClient      *dbent.Client
	db             *sql.DB
	settingService *SettingService
}

func NewLeaderboardService(entClient *dbent.Client, db *sql.DB, settingService *SettingService) *LeaderboardService {
	return &LeaderboardService{
		entClient:      entClient,
		db:             db,
		settingService: settingService,
	}
}

func (s *LeaderboardService) includeAdmin(ctx context.Context) bool {
	return s.settingService.GetLeaderboardSettings(ctx).LeaderboardIncludeAdmin
}

func (s *LeaderboardService) GetBalanceLeaderboard(ctx context.Context, page, pageSize int) (*LeaderboardResult, error) {
	offset := (page - 1) * pageSize

	predicates := []predicate.User{
		dbuser.DeletedAtIsNil(),
		dbuser.StatusEQ(StatusActive),
		dbuser.BalanceGT(0),
	}
	if !s.includeAdmin(ctx) {
		predicates = append(predicates, dbuser.RoleNEQ("admin"))
	}
	total, err := s.entClient.User.Query().
		Where(predicates...).
		Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("count users: %w", err)
	}

	users, err := s.entClient.User.Query().
		Where(predicates...).
		Order(dbent.Desc(dbuser.FieldBalance), dbent.Asc(dbuser.FieldID)).
		Offset(offset).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("query users: %w", err)
	}

	userIDs := make([]int64, 0, len(users))
	for _, user := range users {
		userIDs = append(userIDs, user.ID)
	}
	checkinCounts, err := s.getCheckinCounts(ctx, userIDs)
	if err != nil {
		return nil, fmt.Errorf("query checkin counts: %w", err)
	}

	entries := make([]LeaderboardEntry, 0, len(users))
	for i, u := range users {
		entries = append(entries, LeaderboardEntry{
			Rank:     offset + i + 1,
			Username: leaderboardUserDisplay(u.Username, u.Email),
			Value:    math.Round(u.Balance*100) / 100,
			ExtraInt: checkinCounts[u.ID],
		})
	}

	return &LeaderboardResult{Entries: entries, Total: int64(total)}, nil
}

func (s *LeaderboardService) getCheckinCounts(ctx context.Context, userIDs []int64) (map[int64]int, error) {
	counts := make(map[int64]int, len(userIDs))
	if len(userIDs) == 0 {
		return counts, nil
	}

	placeholders := make([]string, len(userIDs))
	args := make([]any, len(userIDs))
	for index, userID := range userIDs {
		placeholders[index] = fmt.Sprintf("$%d", index+1)
		args[index] = userID
	}
	query := `SELECT user_id, COUNT(*) FROM checkins WHERE user_id IN (` + strings.Join(placeholders, ",") + `) GROUP BY user_id`
	rows, err := s.db.QueryContext(ctx, query, args...)
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

func (s *LeaderboardService) GetConsumptionLeaderboard(ctx context.Context, period string, page, pageSize int) (*LeaderboardResult, error) {
	today := timezone.Today()
	var startTime time.Time
	switch period {
	case "daily":
		startTime = today
	case "weekly":
		startTime = today.AddDate(0, 0, -7)
	case "monthly":
		startTime = today.AddDate(0, 0, -30)
	default:
		startTime = today
	}

	offset := (page - 1) * pageSize

	roleClause := " AND u.role != 'admin'"
	if s.includeAdmin(ctx) {
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
	if err := s.db.QueryRowContext(ctx, countQuery, startTime).Scan(&total); err != nil {
		return nil, fmt.Errorf("count consumption: %w", err)
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
	rows, err := s.db.QueryContext(ctx, dataQuery, startTime, pageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("query consumption: %w", err)
	}
	defer rows.Close()

	entries := make([]LeaderboardEntry, 0)
	rank := offset
	for rows.Next() {
		rank++
		var username, email string
		var totalCost float64
		var requestCount int
		if err := rows.Scan(&username, &email, &totalCost, &requestCount); err != nil {
			return nil, fmt.Errorf("scan consumption row: %w", err)
		}
		entries = append(entries, LeaderboardEntry{
			Rank:     rank,
			Username: leaderboardUserDisplay(username, email),
			Value:    math.Round(totalCost*100) / 100,
			ExtraInt: requestCount,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate consumption rows: %w", err)
	}

	return &LeaderboardResult{Entries: entries, Total: total}, nil
}

func (s *LeaderboardService) GetCheckinLeaderboard(ctx context.Context, page, pageSize int) (*LeaderboardResult, error) {
	today := timezone.Today()
	yesterday := today.AddDate(0, 0, -1)

	offset := (page - 1) * pageSize

	roleClause := " AND u.role != 'admin'"
	if s.includeAdmin(ctx) {
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
	if err := s.db.QueryRowContext(ctx, countQuery, yesterday).Scan(&total); err != nil {
		return nil, fmt.Errorf("count checkin: %w", err)
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
	rows, err := s.db.QueryContext(ctx, dataQuery, yesterday, pageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("query checkin: %w", err)
	}
	defer rows.Close()

	entries := make([]LeaderboardEntry, 0)
	rank := offset
	for rows.Next() {
		rank++
		var username, email string
		var streakDays int
		var rewardAmount float64
		var totalCheckins int
		var lastDate time.Time
		if err := rows.Scan(&username, &email, &streakDays, &rewardAmount, &totalCheckins, &lastDate); err != nil {
			return nil, fmt.Errorf("scan checkin row: %w", err)
		}
		entries = append(entries, LeaderboardEntry{
			Rank:       rank,
			Username:   leaderboardUserDisplay(username, email),
			Value:      float64(streakDays),
			ExtraInt:   totalCheckins,
			ExtraFloat: math.Round(rewardAmount*100) / 100,
			ExtraDate:  lastDate.Format("2006-01-02"),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate checkin rows: %w", err)
	}

	return &LeaderboardResult{Entries: entries, Total: total}, nil
}

func (s *LeaderboardService) GetTransferLeaderboard(ctx context.Context, period string, page, pageSize int) (*LeaderboardResult, error) {
	now := time.Now()
	var start time.Time
	switch period {
	case "weekly":
		start = now.AddDate(0, 0, -7)
	case "monthly":
		start = now.AddDate(0, -1, 0)
	default:
		start = now.AddDate(0, 0, -1)
	}
	roleClause := " AND u.role != 'admin'"
	if s.includeAdmin(ctx) {
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
	if err := s.db.QueryRowContext(ctx, countQuery, start, now).Scan(&total); err != nil {
		return nil, fmt.Errorf("count transfer leaderboard: %w", err)
	}
	offset := (page - 1) * pageSize
	dataQuery := `SELECT u.username, u.email, SUM(bt.amount), COUNT(*)
		FROM balance_transfers bt
		JOIN users u ON u.id = bt.sender_id AND u.deleted_at IS NULL
		WHERE bt.status = 'completed' AND bt.transfer_type = 'direct'
		  AND bt.created_at >= $1 AND bt.created_at < $2 AND u.status = 'active'` + roleClause + `
		GROUP BY bt.sender_id, u.username, u.email
		ORDER BY SUM(bt.amount) DESC, bt.sender_id ASC LIMIT $3 OFFSET $4`
	rows, err := s.db.QueryContext(ctx, dataQuery, start, now, pageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("query transfer leaderboard: %w", err)
	}
	defer rows.Close()
	entries := make([]LeaderboardEntry, 0, pageSize)
	for rows.Next() {
		var username, email string
		var amount float64
		var count int
		if err := rows.Scan(&username, &email, &amount, &count); err != nil {
			return nil, fmt.Errorf("scan transfer leaderboard: %w", err)
		}
		entries = append(entries, LeaderboardEntry{
			Rank: offset + len(entries) + 1, Username: leaderboardUserDisplay(username, email),
			Value: math.Round(amount*1e8) / 1e8, ExtraInt: count,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &LeaderboardResult{Entries: entries, Total: total}, nil
}

func leaderboardUserDisplay(username, email string) string {
	return UserDisplayName(username, email, 0)
}
