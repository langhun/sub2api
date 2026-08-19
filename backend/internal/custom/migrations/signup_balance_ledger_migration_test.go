package migrations

import (
	"strings"
	"testing"
)

func TestSignupBalanceLedgerMigrationIsAppendOnlyAndAuditable(t *testing.T) {
	content, err := files.ReadFile("011_signup_balance_ledger.sql")
	if err != nil {
		t.Fatalf("read signup balance migration: %v", err)
	}
	sql := string(content)
	for _, expected := range []string{
		"CREATE TABLE IF NOT EXISTS custom_signup_balance_ledger",
		"user_id BIGINT NOT NULL UNIQUE REFERENCES users(id) ON DELETE RESTRICT",
		"amount NUMERIC(20, 8) NOT NULL CHECK (amount > 0)",
		"custom_signup_balance_ledger_user_created_idx",
	} {
		if !strings.Contains(sql, expected) {
			t.Fatalf("migration missing %q", expected)
		}
	}
	if strings.Contains(sql, "ALTER TABLE users") || strings.Contains(sql, "DROP TABLE") {
		t.Fatal("signup balance migration must not alter user history or drop tables")
	}
}
