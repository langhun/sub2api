package repository

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// AssignProxyIDsIfUnassigned binds accounts to proxies without overwriting concurrent bindings.
func (r *accountRepository) AssignProxyIDsIfUnassigned(
	ctx context.Context,
	assignments map[int64]int64,
) (map[int64]bool, error) {
	applied := make(map[int64]bool, len(assignments))
	if len(assignments) == 0 {
		return applied, nil
	}

	changedIDs := make([]int64, 0, len(assignments))
	client := clientFromContext(ctx, r.client)
	for accountID, proxyID := range assignments {
		result, err := client.ExecContext(ctx, `
			UPDATE accounts
			SET proxy_id = $1, updated_at = NOW()
			WHERE id = $2 AND proxy_id IS NULL AND deleted_at IS NULL
		`, proxyID, accountID)
		if err != nil {
			return nil, err
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return nil, err
		}
		if rows > 0 {
			applied[accountID] = true
			changedIDs = append(changedIDs, accountID)
		}
	}

	if len(changedIDs) > 0 {
		payload := map[string]any{"account_ids": changedIDs}
		err := enqueueSchedulerOutbox(ctx, r.sql, service.SchedulerOutboxEventAccountBulkChanged, nil, nil, payload)
		if err != nil {
			logger.LegacyPrintf("repository.account", "[SchedulerOutbox] enqueue proxy assignment failed: err=%v", err)
		}
	}
	return applied, nil
}
