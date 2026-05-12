package service

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/predicate"
	dbuser "github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/lib/pq"
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

type LeaderboardSummary struct {
	TotalValue float64 `json:"total_value"`
	TotalUsers int64   `json:"total_users"`
}

type LeaderboardChartItem struct {
	Username string  `json:"username"`
	Value    float64 `json:"value"`
}

type LeaderboardResult struct {
	Entries    []LeaderboardEntry     `json:"items"`
	Total      int64                  `json:"total"`
	Summary    *LeaderboardSummary    `json:"summary,omitempty"`
	ChartItems []LeaderboardChartItem `json:"chart_items,omitempty"`
}

type LeaderboardService struct {
	entClient *dbent.Client
	db        *sql.DB
}

func NewLeaderboardService(entClient *dbent.Client, db *sql.DB) *LeaderboardService {
	return &LeaderboardService{
		entClient: entClient,
		db:        db,
	}
}

func (s *LeaderboardService) GetBalanceLeaderboard(ctx context.Context, page, pageSize int, includeAdmin bool) (*LeaderboardResult, error) {
	offset := (page - 1) * pageSize

	filters := []predicate.User{
		dbuser.DeletedAtIsNil(),
		dbuser.StatusEQ(StatusActive),
		dbuser.BalanceGT(0),
	}
	if !includeAdmin {
		filters = append(filters, dbuser.RoleNEQ("admin"))
	}

	total, err := s.entClient.User.Query().
		Where(filters...).
		Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("count users: %w", err)
	}

	users, err := s.entClient.User.Query().
		Where(filters...).
		Order(dbent.Desc(dbuser.FieldBalance)).
		Offset(offset).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("query users: %w", err)
	}

	userIDs := make([]int64, 0, len(users))
	for _, u := range users {
		userIDs = append(userIDs, u.ID)
	}
	checkinCounts, err := s.getCheckinCounts(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	entries := make([]LeaderboardEntry, 0, len(users))
	for i, u := range users {
		entries = append(entries, LeaderboardEntry{
			Rank:     offset + i + 1,
			Username: maskUsername(u.Username, u.Email),
			Value:    math.Round(u.Balance*100) / 100,
			ExtraInt: checkinCounts[u.ID],
		})
	}

	return &LeaderboardResult{Entries: entries, Total: int64(total)}, nil
}

func (s *LeaderboardService) GetConsumptionLeaderboard(ctx context.Context, period string, page, pageSize int, includeAdmin bool) (*LeaderboardResult, error) {
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
	roleFilter := ""
	if !includeAdmin {
		roleFilter = " AND u.role != 'admin'"
	}

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) FROM (
			SELECT ul.user_id
			FROM usage_logs ul
			INNER JOIN users u ON ul.user_id = u.id AND u.deleted_at IS NULL
			WHERE ul.created_at >= $1 AND u.status = 'active'%s
			GROUP BY ul.user_id
			HAVING SUM(ul.actual_cost) > 0
		) sub
	`, roleFilter)
	var total int64
	if err := s.db.QueryRowContext(ctx, countQuery, startTime).Scan(&total); err != nil {
		return nil, fmt.Errorf("count consumption: %w", err)
	}

	dataQuery := fmt.Sprintf(`
		SELECT u.username, u.email, COALESCE(SUM(ul.actual_cost), 0) as total_cost, COUNT(*) as request_count
		FROM usage_logs ul
		INNER JOIN users u ON ul.user_id = u.id AND u.deleted_at IS NULL
		WHERE ul.created_at >= $1 AND u.status = 'active'%s
		GROUP BY ul.user_id, u.username, u.email
		HAVING SUM(ul.actual_cost) > 0
		ORDER BY total_cost DESC, ul.user_id ASC
		LIMIT $2 OFFSET $3
	`, roleFilter)
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
			Username: maskUsername(username, email),
			Value:    math.Round(totalCost*100) / 100,
			ExtraInt: requestCount,
		})
	}

	chartQuery := fmt.Sprintf(`
		SELECT u.username, u.email, COALESCE(SUM(ul.actual_cost), 0) as total_cost
		FROM usage_logs ul
		INNER JOIN users u ON ul.user_id = u.id AND u.deleted_at IS NULL
		WHERE ul.created_at >= $1 AND u.status = 'active'%s
		GROUP BY ul.user_id, u.username, u.email
		HAVING SUM(ul.actual_cost) > 0
		ORDER BY total_cost DESC, ul.user_id ASC
	`, roleFilter)
	chartRows, err := s.db.QueryContext(ctx, chartQuery, startTime)
	if err != nil {
		return nil, fmt.Errorf("query consumption chart: %w", err)
	}
	defer chartRows.Close()

	chartItems := make([]LeaderboardChartItem, 0, total)
	var totalValue float64
	for chartRows.Next() {
		var username, email string
		var totalCost float64
		if err := chartRows.Scan(&username, &email, &totalCost); err != nil {
			return nil, fmt.Errorf("scan consumption chart row: %w", err)
		}
		totalValue += totalCost
		chartItems = append(chartItems, LeaderboardChartItem{
			Username: maskUsername(username, email),
			Value:    math.Round(totalCost*100) / 100,
		})
	}
	if err := chartRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate consumption chart rows: %w", err)
	}

	return &LeaderboardResult{
		Entries: entries,
		Total:   total,
		Summary: &LeaderboardSummary{
			TotalValue: math.Round(totalValue*100) / 100,
			TotalUsers: total,
		},
		ChartItems: chartItems,
	}, nil
}

func (s *LeaderboardService) GetCheckinLeaderboard(ctx context.Context, page, pageSize int, includeAdmin bool) (*LeaderboardResult, error) {
	today := timezone.Today()
	yesterday := today.AddDate(0, 0, -1)

	offset := (page - 1) * pageSize
	roleFilter := ""
	if !includeAdmin {
		roleFilter = " AND u.role != 'admin'"
	}

	dataQuery := fmt.Sprintf(`
		WITH latest_checkins AS (
			SELECT user_id, MAX(checkin_date) AS last_date, COUNT(*) AS total_checkins
			FROM checkins
			GROUP BY user_id
		),
		eligible_checkins AS (
			SELECT u.username, u.email, c.streak_days, c.reward_amount,
				latest_checkins.total_checkins, latest_checkins.last_date
			FROM latest_checkins
			INNER JOIN checkins c ON c.user_id = latest_checkins.user_id AND c.checkin_date = latest_checkins.last_date
			INNER JOIN users u ON latest_checkins.user_id = u.id AND u.deleted_at IS NULL
			WHERE latest_checkins.last_date >= $1 AND u.status = 'active'%s
		),
		page_checkins AS (
			SELECT *
			FROM eligible_checkins
			ORDER BY streak_days DESC, last_date DESC
			LIMIT $2 OFFSET $3
		)
		SELECT p.username, p.email, p.streak_days, p.reward_amount,
			p.total_checkins, p.last_date, total.total_count
		FROM (SELECT COUNT(*) AS total_count FROM eligible_checkins) total
		LEFT JOIN page_checkins p ON TRUE
		ORDER BY p.streak_days DESC NULLS LAST, p.last_date DESC NULLS LAST
	`, roleFilter)
	rows, err := s.db.QueryContext(ctx, dataQuery, yesterday, pageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("query checkin: %w", err)
	}
	defer rows.Close()

	entries := make([]LeaderboardEntry, 0)
	var total int64
	rank := offset
	for rows.Next() {
		var username, email sql.NullString
		var streakDays sql.NullInt64
		var rewardAmount sql.NullFloat64
		var totalCheckins sql.NullInt64
		var lastDate sql.NullTime
		var rowTotal int64
		if err := rows.Scan(&username, &email, &streakDays, &rewardAmount, &totalCheckins, &lastDate, &rowTotal); err != nil {
			return nil, fmt.Errorf("scan checkin row: %w", err)
		}
		total = rowTotal
		if !streakDays.Valid {
			continue
		}
		rank++
		entries = append(entries, LeaderboardEntry{
			Rank:       rank,
			Username:   maskUsername(username.String, email.String),
			Value:      float64(streakDays.Int64),
			ExtraInt:   int(totalCheckins.Int64),
			ExtraFloat: math.Round(rewardAmount.Float64*100) / 100,
			ExtraDate:  lastDate.Time.Format("2006-01-02"),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate checkin rows: %w", err)
	}

	return &LeaderboardResult{Entries: entries, Total: total}, nil
}

func (s *LeaderboardService) getCheckinCounts(ctx context.Context, userIDs []int64) (map[int64]int, error) {
	counts := make(map[int64]int, len(userIDs))
	if len(userIDs) == 0 {
		return counts, nil
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT user_id, COUNT(*)
		FROM checkins
		WHERE user_id = ANY($1)
		GROUP BY user_id
	`, pq.Array(userIDs))
	if err != nil {
		return nil, fmt.Errorf("query checkin counts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var userID int64
		var count int
		if err := rows.Scan(&userID, &count); err != nil {
			return nil, fmt.Errorf("scan checkin count row: %w", err)
		}
		counts[userID] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate checkin count rows: %w", err)
	}
	return counts, nil
}

func maskUsername(username, email string) string {
	if username != "" {
		return username
	}
	if email == "" {
		return "user"
	}
	if len(email) <= 3 {
		return email
	}
	return email[:3] + "***"
}
