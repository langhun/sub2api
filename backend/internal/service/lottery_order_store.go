package service

import (
	"context"
	"database/sql"
	"fmt"
	"hash/fnv"
	"strings"

	dbent "github.com/Wei-Shaw/sub2api/ent"
)

func runSerializableLotteryTx(
	ctx context.Context,
	client *dbent.Client,
	fn func(*dbent.Client) (*LotteryBetResult, error),
) (result *LotteryBetResult, err error) {
	err = runSerializableLotteryOperationTx(ctx, client, func(txClient *dbent.Client) error {
		var fnErr error
		result, fnErr = fn(txClient)
		return fnErr
	})
	return result, err
}

func runSerializableLotteryOperationTx(
	ctx context.Context,
	client *dbent.Client,
	fn func(*dbent.Client) error,
) (err error) {
	if client == nil {
		return ErrLotteryStorageUnavailable
	}
	var lastErr error
	for attempt := 0; attempt < bankTxMaxRetries; attempt++ {
		tx, beginErr := client.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
		if beginErr != nil {
			return fmt.Errorf("begin lottery transaction: %w", beginErr)
		}
		committed := false
		func() {
			defer func() {
				if recovered := recover(); recovered != nil {
					_ = tx.Rollback()
					err = fmt.Errorf("lottery transaction panic: %v", recovered)
					return
				}
				if !committed && err != nil {
					_ = tx.Rollback()
				}
			}()
			err = fn(tx.Client())
			if err != nil {
				return
			}
			if commitErr := tx.Commit(); commitErr != nil {
				err = fmt.Errorf("commit lottery transaction: %w", commitErr)
				return
			}
			committed = true
		}()
		if err == nil {
			return nil
		}
		if isRetryableBankTxErr(err) {
			lastErr = err
			continue
		}
		return err
	}
	return lastErr
}

func lockLotteryIssueScope(ctx context.Context, client *dbent.Client, lotteryType, issueNo string, userID int64) error {
	rows, err := client.QueryContext(ctx, "SELECT pg_advisory_xact_lock($1)", lotteryIssueScopeLockID(lotteryType, issueNo, userID))
	if err != nil {
		return fmt.Errorf("lock lottery issue scope: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return rows.Err()
}

func lotteryIssueScopeLockID(lotteryType, issueNo string, userID int64) int64 {
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(fmt.Sprintf("%s:%s:%d", lotteryType, issueNo, userID)))
	return int64(hasher.Sum64())
}

func ensureLotteryIssueInTx(ctx context.Context, client *dbent.Client, issue *Issue) error {
	if issue == nil {
		return ErrLotteryDataInvalid
	}
	if _, err := client.ExecContext(ctx, `
INSERT INTO lottery_issue (lottery_type, issue_no, open_time, status, created_at, updated_at)
VALUES ($1, $2, $3, $4, NOW(), NOW())
ON CONFLICT (lottery_type, issue_no) DO UPDATE
SET open_time = EXCLUDED.open_time,
    status = EXCLUDED.status,
    updated_at = NOW()
`, issue.LotteryType, issue.IssueNo, issue.OpenTime, issue.Status); err != nil {
		return fmt.Errorf("ensure lottery issue: %w", err)
	}
	return nil
}

func countLotteryOrdersByUserIssue(ctx context.Context, client *dbent.Client, userID int64, lotteryType, issueNo string) (int, error) {
	rows, err := client.QueryContext(ctx, `
SELECT COUNT(*)
FROM lottery_order
WHERE user_id = $1
  AND lottery_type = $2
  AND issue_no = $3
`, userID, lotteryType, issueNo)
	if err != nil {
		return 0, fmt.Errorf("count lottery orders: %w", err)
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return 0, fmt.Errorf("count lottery orders rows: %w", err)
		}
		return 0, sql.ErrNoRows
	}
	var count int
	if err := rows.Scan(&count); err != nil {
		return 0, fmt.Errorf("scan lottery order count: %w", err)
	}
	return count, rows.Err()
}

func findLotteryOrderByNumbers(ctx context.Context, client *dbent.Client, payload lotteryBetPayload) (*lotteryOrderRecord, error) {
	rows, err := client.QueryContext(ctx, `
SELECT id, lottery_type, issue_no, red_balls, blue_ball, cost, reward, prize_level, red_hits, blue_hit, status, created_at
FROM lottery_order
WHERE user_id = $1
  AND lottery_type = $2
  AND issue_no = $3
  AND red_balls = $4
  AND blue_ball = $5
ORDER BY id DESC
LIMIT 1
`, payload.userID, payload.lotteryType, payload.issueNo, strings.Join(payload.redBalls, ","), payload.blueBall)
	if err != nil {
		return nil, fmt.Errorf("find lottery order: %w", err)
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("find lottery order rows: %w", err)
		}
		return nil, nil
	}
	record, err := scanLotteryOrderRecord(rows)
	if err != nil {
		return nil, err
	}
	return record, rows.Err()
}

func createLotteryOrderInTx(ctx context.Context, client *dbent.Client, payload lotteryBetPayload) (int64, error) {
	rows, err := client.QueryContext(ctx, `
INSERT INTO lottery_order (
    lottery_type,
    issue_no,
    user_id,
    red_balls,
    blue_ball,
    cost,
    reward,
    prize_level,
    red_hits,
    blue_hit,
    status,
    created_at,
    updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, 0, '', 0, FALSE, $7, NOW(), NOW())
RETURNING id
`, payload.lotteryType, payload.issueNo, payload.userID, strings.Join(payload.redBalls, ","), payload.blueBall, lotterySingleBetCost, lotteryOrderStatusPending)
	if err != nil {
		return 0, fmt.Errorf("create lottery order: %w", err)
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return 0, fmt.Errorf("create lottery order rows: %w", err)
		}
		return 0, sql.ErrNoRows
	}
	var orderID int64
	if err := rows.Scan(&orderID); err != nil {
		return 0, fmt.Errorf("scan lottery order id: %w", err)
	}
	return orderID, rows.Err()
}

func listLotteryOrders(ctx context.Context, client *dbent.Client, query LotteryOrderQuery) ([]LotteryOrderView, error) {
	sqlText := `
SELECT id, lottery_type, issue_no, red_balls, blue_ball, cost, reward, prize_level, red_hits, blue_hit, status, created_at
FROM lottery_order
WHERE user_id = $1
`
	args := []any{query.UserID}
	if query.LotteryType != "" {
		sqlText += " AND lottery_type = $2"
		args = append(args, query.LotteryType)
		if query.IssueNo != "" {
			sqlText += " AND issue_no = $3"
			args = append(args, query.IssueNo)
		}
	} else if query.IssueNo != "" {
		sqlText += " AND issue_no = $2"
		args = append(args, query.IssueNo)
	}
	sqlText += " ORDER BY created_at DESC, id DESC"

	rows, err := client.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return nil, fmt.Errorf("list lottery orders: %w", err)
	}
	defer func() { _ = rows.Close() }()
	views := make([]LotteryOrderView, 0)
	for rows.Next() {
		record, err := scanLotteryOrderRecord(rows)
		if err != nil {
			return nil, err
		}
		view := LotteryOrderView{
			ID:          record.ID,
			LotteryType: record.LotteryType,
			IssueNo:     record.IssueNo,
			RedBalls:    append([]string(nil), record.RedBalls...),
			BlueBall:    record.BlueBall,
			Cost:        record.Cost,
			Reward:      record.Reward,
			PrizeLevel:  record.PrizeLevel,
			RedHits:     record.RedHits,
			BlueHit:     record.BlueHit,
			Status:      record.Status,
		}
		if record.CreatedAt.Valid {
			view.CreatedAt = record.CreatedAt.Time
		}
		views = append(views, view)
	}
	return views, rows.Err()
}

func scanLotteryOrderRecord(scanner interface{ Scan(dest ...any) error }) (*lotteryOrderRecord, error) {
	var record lotteryOrderRecord
	var redBalls string
	if err := scanner.Scan(
		&record.ID,
		&record.LotteryType,
		&record.IssueNo,
		&redBalls,
		&record.BlueBall,
		&record.Cost,
		&record.Reward,
		&record.PrizeLevel,
		&record.RedHits,
		&record.BlueHit,
		&record.Status,
		&record.CreatedAt,
	); err != nil {
		return nil, fmt.Errorf("scan lottery order record: %w", err)
	}
	record.RedBalls = strings.Split(strings.TrimSpace(redBalls), ",")
	return &record, nil
}
