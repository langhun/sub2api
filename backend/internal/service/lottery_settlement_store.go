package service

import (
	"context"
	"fmt"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/shopspring/decimal"
)

type lotterySettlementOrder struct {
	ID          int64
	LotteryType string
	IssueNo     string
	UserID      int64
	RedBalls    []string
	BlueBall    string
	Cost        decimal.Decimal
}

func getLotteryIssueStatusForUpdate(ctx context.Context, client *dbent.Client, lotteryType, issueNo string) (string, bool, error) {
	rows, err := client.QueryContext(ctx, `
SELECT status
FROM lottery_issue
WHERE lottery_type = $1
  AND issue_no = $2
FOR UPDATE
`, lotteryType, issueNo)
	if err != nil {
		return "", false, fmt.Errorf("query lottery issue status: %w", err)
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return "", false, fmt.Errorf("query lottery issue status rows: %w", err)
		}
		return "", false, nil
	}
	status, err := scanLotteryIssueStatus(rows)
	if err != nil {
		return "", false, fmt.Errorf("scan lottery issue status: %w", err)
	}
	return status, true, rows.Err()
}

func listPendingLotteryOrdersForSettlement(ctx context.Context, client *dbent.Client, lotteryType, issueNo string) ([]lotterySettlementOrder, error) {
	rows, err := client.QueryContext(ctx, `
SELECT id, lottery_type, issue_no, user_id, red_balls, blue_ball, cost
FROM lottery_order
WHERE lottery_type = $1
  AND issue_no = $2
  AND status = $3
ORDER BY id ASC
FOR UPDATE
`, lotteryType, issueNo, lotteryOrderStatusPending)
	if err != nil {
		return nil, fmt.Errorf("query lottery settlement orders: %w", err)
	}
	defer func() { _ = rows.Close() }()

	orders := make([]lotterySettlementOrder, 0)
	for rows.Next() {
		order, err := scanLotterySettlementOrder(rows)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	return orders, rows.Err()
}

func scanLotterySettlementOrder(scanner interface{ Scan(dest ...any) error }) (lotterySettlementOrder, error) {
	var order lotterySettlementOrder
	var redBalls string
	if err := scanner.Scan(
		&order.ID,
		&order.LotteryType,
		&order.IssueNo,
		&order.UserID,
		&redBalls,
		&order.BlueBall,
		&order.Cost,
	); err != nil {
		return lotterySettlementOrder{}, fmt.Errorf("scan lottery settlement order: %w", err)
	}
	order.RedBalls = splitLotteryBallList(redBalls)
	return order, nil
}

func updateLotteryOrderSettlementInTx(ctx context.Context, client *dbent.Client, order lotterySettlementOrder, prize lotteryPrize) error {
	status := lotteryOrderStatusLose
	if prize.reward.GreaterThan(decimal.Zero) {
		status = lotteryOrderStatusWin
	}
	result, err := client.ExecContext(ctx, `
UPDATE lottery_order
SET reward = $2,
    prize_level = $3,
    red_hits = $4,
    blue_hit = $5,
    status = $6,
    settled_at = NOW(),
    updated_at = NOW()
WHERE id = $1
  AND status = $7
`, order.ID, prize.reward, prize.level, prize.redHits, prize.blueHit, status, lotteryOrderStatusPending)
	if err != nil {
		return fmt.Errorf("update lottery settlement order: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("update lottery settlement rows affected: %w", err)
	}
	if affected != 1 {
		return ErrLotteryResultConflict.WithMetadata(map[string]string{
			"order_id": fmt.Sprintf("%d", order.ID),
			"issue_no": order.IssueNo,
		})
	}
	return nil
}

func createLotteryRewardLogInTx(ctx context.Context, client *dbent.Client, order lotterySettlementOrder, prize lotteryPrize) error {
	remark := fmt.Sprintf("lottery %s prize for issue %s", prize.level, order.IssueNo)
	if _, err := client.ExecContext(ctx, `
INSERT INTO lottery_reward_log (
    lottery_type,
    user_id,
    issue_no,
    order_id,
    reward,
    remark,
    created_at
)
VALUES ($1, $2, $3, $4, $5, $6, NOW())
`, order.LotteryType, order.UserID, order.IssueNo, order.ID, prize.reward, remark); err != nil {
		return fmt.Errorf("create lottery reward log: %w", err)
	}
	return nil
}

func markLotteryIssueSettledInTx(ctx context.Context, client *dbent.Client, lotteryType, issueNo string) error {
	result, err := client.ExecContext(ctx, `
UPDATE lottery_issue
SET status = $3,
    settled_at = COALESCE(settled_at, NOW()),
    updated_at = NOW()
WHERE lottery_type = $1
  AND issue_no = $2
  AND status <> $3
`, lotteryType, issueNo, lotteryIssueStatusSettled)
	if err != nil {
		return fmt.Errorf("mark lottery issue settled: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("mark lottery issue settled rows affected: %w", err)
	}
	if affected > 1 {
		return fmt.Errorf("mark lottery issue settled affected %d rows", affected)
	}
	return nil
}
