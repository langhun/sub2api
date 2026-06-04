package service

import (
	"context"
	"database/sql"
	"fmt"

	dbent "github.com/Wei-Shaw/sub2api/ent"
)

func listOpenedLotteryIssues(ctx context.Context, client *dbent.Client, lotteryType string, limit int) ([]Issue, error) {
	rows, err := client.QueryContext(ctx, `
SELECT lottery_type, issue_no, open_time, status
FROM lottery_issue
WHERE lottery_type = $1
  AND status = $2
  AND settled_at IS NULL
ORDER BY open_time ASC, id ASC
LIMIT $3
`, lotteryType, lotteryIssueStatusOpened, limit)
	if err != nil {
		return nil, fmt.Errorf("list opened lottery issues: %w", err)
	}
	defer func() { _ = rows.Close() }()

	issues := make([]Issue, 0)
	for rows.Next() {
		issue, err := scanLotteryIssue(rows)
		if err != nil {
			return nil, err
		}
		issues = append(issues, issue)
	}
	return issues, rows.Err()
}

func scanLotteryIssue(scanner interface{ Scan(dest ...any) error }) (Issue, error) {
	var issue Issue
	var openTime sql.NullTime
	if err := scanner.Scan(
		&issue.LotteryType,
		&issue.IssueNo,
		&openTime,
		&issue.Status,
	); err != nil {
		return Issue{}, fmt.Errorf("scan lottery issue: %w", err)
	}
	if openTime.Valid {
		issue.OpenTime = openTime.Time
		issue.CutoffTime = issue.OpenTime.Add(-lotteryCutoffLead)
	}
	return issue, nil
}
