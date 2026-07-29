package migrations

import (
	"context"
	"strings"
	"testing"
)

func TestApplyRejectsNilDatabase(t *testing.T) {
	if err := Apply(context.Background(), nil); err == nil || err.Error() != "nil sql db" {
		t.Fatalf("Apply(nil) error = %v, want nil sql db", err)
	}
}

func TestActivityRedeemMetadataMigrationPreservesLegacyColumns(t *testing.T) {
	content, err := files.ReadFile("001_activity_redeem_metadata.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	sql := string(content)
	for _, expected := range []string{
		"CREATE TABLE IF NOT EXISTS custom_activity_redeem_metadata",
		"REFERENCES redeem_codes(id) ON DELETE CASCADE",
		"SELECT id, multiplier, bet_amount",
		"ON CONFLICT (redeem_code_id) DO NOTHING",
	} {
		if !strings.Contains(sql, expected) {
			t.Fatalf("migration missing %q", expected)
		}
	}
	if strings.Contains(sql, "ALTER TABLE redeem_codes") || strings.Contains(sql, "DROP COLUMN") {
		t.Fatal("custom migration must not alter immutable legacy redeem_codes columns")
	}
}

func TestAccountDrainMigrationOwnsOnlyCustomTables(t *testing.T) {
	content, err := files.ReadFile("002_account_drain_plans.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	sql := string(content)
	for _, expected := range []string{
		"CREATE TABLE IF NOT EXISTS custom_account_drain_plans",
		"CREATE TABLE IF NOT EXISTS custom_account_drain_plan_accounts",
		"REFERENCES accounts(id) ON DELETE CASCADE",
	} {
		if !strings.Contains(sql, expected) {
			t.Fatalf("migration missing %q", expected)
		}
	}
	if strings.Contains(sql, "ALTER TABLE accounts") || strings.Contains(sql, "DROP TABLE accounts") {
		t.Fatal("custom account drain migration must not modify upstream accounts")
	}
}
