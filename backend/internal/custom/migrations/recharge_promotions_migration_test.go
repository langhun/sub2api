package migrations

import (
	"strings"
	"testing"
)

func TestRechargePromotionMigrationOwnsOnlyCustomSnapshotTable(t *testing.T) {
	content, err := files.ReadFile("003_recharge_promotions.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	sql := string(content)
	for _, expected := range []string{
		"CREATE TABLE IF NOT EXISTS custom_recharge_promotion_orders",
		"payment_order_id BIGINT NOT NULL UNIQUE",
		"base_credit NUMERIC",
		"bonus_credit NUMERIC",
		"final_credit NUMERIC",
		"custom_recharge_promotion_orders_total_check",
	} {
		if !strings.Contains(sql, expected) {
			t.Fatalf("migration missing %q", expected)
		}
	}
	if strings.Contains(sql, "ALTER TABLE payment_orders") || strings.Contains(sql, "DROP TABLE") {
		t.Fatal("custom recharge promotion migration must not alter upstream order schema")
	}
}
