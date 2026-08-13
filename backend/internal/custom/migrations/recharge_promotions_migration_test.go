package migrations

import (
	"strings"
	"testing"
)

func TestRechargePromotionsMigrationOwnsOnlyCustomTables(t *testing.T) {
	base, err := files.ReadFile("003_recharge_promotions.sql")
	if err != nil {
		t.Fatalf("read base migration: %v", err)
	}
	canonical, compatible := migrationChecksums(string(base))
	const productionChecksum = "fd089a8d525e32c7368d0e4e585c922b87a2cd90b6b482becbe19b5166b5b707"
	if canonical != productionChecksum {
		t.Fatalf("base migration checksum = %s, want production checksum %s", canonical, productionChecksum)
	}
	if _, ok := compatible[productionChecksum]; !ok {
		t.Fatal("base migration no longer accepts the production checksum")
	}

	upgrade, err := files.ReadFile("004_recharge_promotions_upgrade.sql")
	if err != nil {
		t.Fatalf("read upgrade migration: %v", err)
	}
	tieredPercentage, err := files.ReadFile("005_recharge_promotions_tiered_percentage.sql")
	if err != nil {
		t.Fatalf("read tiered percentage migration: %v", err)
	}
	sql := string(base) + "\n" + string(upgrade) + "\n" + string(tieredPercentage)
	for _, expected := range []string{
		"CREATE TABLE IF NOT EXISTS custom_recharge_promotion_orders",
		"RENAME COLUMN bonus_rate TO percentage_rate",
		"FOREIGN KEY (payment_order_id) REFERENCES payment_orders(id)",
		"tier_rules JSONB NOT NULL",
		"refunded_bonus_credit",
		"refunded_credit = refunded_base_credit + refunded_bonus_credit",
		"custom_recharge_promotion_orders_user_created_idx",
		"tiered_percentage_rules JSONB NOT NULL",
		"matched_minimum_amount",
		"matched_maximum_amount",
		"applied_percentage_rate",
		"'tiered_percentage'",
	} {
		if !strings.Contains(sql, expected) {
			t.Fatalf("migration missing %q", expected)
		}
	}
	if strings.Contains(sql, "ALTER TABLE payment_orders") || strings.Contains(sql, "DROP TABLE") || strings.Contains(sql, "DELETE FROM") {
		t.Fatal("recharge promotions migration must not mutate upstream tables")
	}
}
