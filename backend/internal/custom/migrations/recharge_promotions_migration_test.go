package migrations

import (
	"strings"
	"testing"
)

func TestRechargePromotionsMigrationOwnsOnlyCustomTables(t *testing.T) {
	content, err := files.ReadFile("003_recharge_promotions.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	sql := string(content)
	for _, expected := range []string{
		"CREATE TABLE IF NOT EXISTS custom_recharge_promotion_orders",
		"payment_order_id BIGINT NOT NULL UNIQUE REFERENCES payment_orders(id)",
		"tier_rules JSONB NOT NULL",
		"refunded_bonus_credit",
		"refunded_credit = refunded_base_credit + refunded_bonus_credit",
		"custom_recharge_promotion_orders_user_created_idx",
	} {
		if !strings.Contains(sql, expected) {
			t.Fatalf("migration missing %q", expected)
		}
	}
	if strings.Contains(sql, "ALTER TABLE payment_orders") || strings.Contains(sql, "DROP TABLE") {
		t.Fatal("recharge promotions migration must not mutate upstream tables")
	}
}
