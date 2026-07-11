package migrations

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigration181CreatesDurableRewardDeliveryOutbox(t *testing.T) {
	content, err := FS.ReadFile("181_create_reward_deliveries.sql")
	require.NoError(t, err)

	sql := string(content)
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS reward_deliveries")
	require.Contains(t, sql, "reward_snapshot JSONB NOT NULL")
	require.Contains(t, sql, "UNIQUE (idempotency_key)")
	require.Contains(t, sql, "UNIQUE (source_type, source_id)")
	require.Contains(t, sql, "source_id > 0")
	require.Contains(t, sql, "user_id > 0")
	require.Contains(t, sql, "reward_value >= 0")
	require.Contains(t, sql, "'pending', 'delivering', 'delivered', 'failed', 'compensated'")
	require.Contains(t, sql, "idx_reward_deliveries_due")
	require.Contains(t, sql, "idx_reward_deliveries_user")
	require.NotContains(t, sql, "DROP TABLE")
}
