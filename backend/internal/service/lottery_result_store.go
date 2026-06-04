package service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	dbent "github.com/Wei-Shaw/sub2api/ent"
)

func saveLotteryResultInTx(ctx context.Context, client *dbent.Client, result *Result) (LotteryResultView, bool, error) {
	existing, err := getLotteryResultByIssueInTx(ctx, client, result.LotteryType, result.IssueNo)
	if err != nil {
		return LotteryResultView{}, false, err
	}
	if existing != nil {
		if lotteryResultNumbersMatch(*existing, result) {
			return *existing, true, nil
		}
		return LotteryResultView{}, false, lotteryResultConflictError(*existing, result)
	}

	rows, err := client.QueryContext(ctx, `
INSERT INTO lottery_result (
    lottery_type,
    issue_no,
    red_balls,
    blue_ball,
    source,
    source_ref,
    source_payload,
    opened_at,
    created_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8, NOW())
RETURNING id, lottery_type, issue_no, red_balls, blue_ball, opened_at, source, source_ref, created_at
`, result.LotteryType, result.IssueNo, strings.Join(result.RedBalls, ","), result.BlueBall, result.Source, result.SourceRef, string(result.SourcePayload), result.OpenedAt)
	if err != nil {
		return LotteryResultView{}, false, fmt.Errorf("insert lottery result: %w", err)
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return LotteryResultView{}, false, fmt.Errorf("insert lottery result rows: %w", err)
		}
		return LotteryResultView{}, false, sql.ErrNoRows
	}
	view, err := scanLotteryResultView(rows)
	if err != nil {
		return LotteryResultView{}, false, err
	}
	return view, false, rows.Err()
}

func getLotteryResultByIssueInTx(ctx context.Context, client *dbent.Client, lotteryType, issueNo string) (*LotteryResultView, error) {
	rows, err := client.QueryContext(ctx, `
SELECT id, lottery_type, issue_no, red_balls, blue_ball, opened_at, source, source_ref, created_at
FROM lottery_result
WHERE lottery_type = $1
  AND issue_no = $2
LIMIT 1
`, lotteryType, issueNo)
	if err != nil {
		return nil, fmt.Errorf("query lottery result: %w", err)
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("query lottery result rows: %w", err)
		}
		return nil, nil
	}
	view, err := scanLotteryResultView(rows)
	if err != nil {
		return nil, err
	}
	return &view, rows.Err()
}

func markLotteryIssueOpenedInTx(ctx context.Context, client *dbent.Client, result *Result) error {
	openTime := lotteryResultOpenTime(result)
	if _, err := client.ExecContext(ctx, `
INSERT INTO lottery_issue (
    lottery_type,
    issue_no,
    open_time,
    status,
    result_synced_at,
    created_at,
    updated_at
)
VALUES ($1, $2, $3, $4, NOW(), NOW(), NOW())
ON CONFLICT (lottery_type, issue_no) DO UPDATE
SET open_time = EXCLUDED.open_time,
    status = CASE
        WHEN lottery_issue.status = 'settled' THEN lottery_issue.status
        ELSE EXCLUDED.status
    END,
    result_synced_at = COALESCE(lottery_issue.result_synced_at, NOW()),
    updated_at = NOW()
`, result.LotteryType, result.IssueNo, openTime, lotteryIssueStatusOpened); err != nil {
		return fmt.Errorf("mark lottery issue opened: %w", err)
	}
	return nil
}

func scanLotteryResultView(scanner interface{ Scan(dest ...any) error }) (LotteryResultView, error) {
	var view LotteryResultView
	var redBalls string
	var openedAt sql.NullTime
	var createdAt sql.NullTime
	if err := scanner.Scan(
		&view.ID,
		&view.LotteryType,
		&view.IssueNo,
		&redBalls,
		&view.BlueBall,
		&openedAt,
		&view.Source,
		&view.SourceRef,
		&createdAt,
	); err != nil {
		return LotteryResultView{}, fmt.Errorf("scan lottery result: %w", err)
	}
	view.RedBalls = splitLotteryBallList(redBalls)
	if openedAt.Valid {
		view.OpenedAt = openedAt.Time
	}
	if createdAt.Valid {
		view.CreatedAt = createdAt.Time
	}
	return view, nil
}

func lotteryResultNumbersMatch(existing LotteryResultView, incoming *Result) bool {
	if incoming == nil {
		return false
	}
	return strings.Join(existing.RedBalls, ",") == strings.Join(incoming.RedBalls, ",") &&
		existing.BlueBall == incoming.BlueBall
}

func splitLotteryBallList(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return []string{}
	}
	return strings.Split(trimmed, ",")
}
