package migrations

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigration185WidensLargeBalanceAmountColumns(t *testing.T) {
	content, err := FS.ReadFile("185_widen_large_balance_amount_columns.sql")
	require.NoError(t, err)

	sql := string(content)
	require.Contains(t, sql, "ALTER COLUMN balance TYPE DECIMAL(38, 18)")
	require.Contains(t, sql, "ALTER COLUMN total_recharged TYPE DECIMAL(38, 18)")
	require.Contains(t, sql, "ALTER COLUMN reward_amount TYPE DECIMAL(38, 18)")
	require.Contains(t, sql, "ALTER COLUMN bet_amount TYPE DECIMAL(38, 18)")
	require.Contains(t, sql, "ALTER COLUMN value TYPE DECIMAL(38, 18)")
}
