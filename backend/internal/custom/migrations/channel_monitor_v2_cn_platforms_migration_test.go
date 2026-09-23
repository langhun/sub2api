package migrations

import (
	"strings"
	"testing"
)

func TestChannelMonitorV2CNPlatformsMigrationAppendsWithoutReplacingConfig(t *testing.T) {
	content, err := files.ReadFile("016_channel_monitor_v2_cn_platforms.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	sql := string(content)

	for _, expected := range []string{
		"FROM channel_monitor_v2_config AS config",
		"jsonb_array_elements(config.platforms)",
		"lower(btrim(existing.item ->> 'platform')) = desired.platform",
		"platforms = config.platforms || additions.items",
		"version = config.version + 1",
		"'enabled', TRUE",
		"'models', '[]'::jsonb",
	} {
		if !strings.Contains(sql, expected) {
			t.Fatalf("migration missing %q", expected)
		}
	}

	kimi := strings.Index(sql, "('kimi', 1)")
	zhipu := strings.Index(sql, "('zhipu', 2)")
	deepseek := strings.Index(sql, "('deepseek', 3)")
	if kimi < 0 || zhipu < 0 || deepseek < 0 || kimi >= zhipu || zhipu >= deepseek {
		t.Fatal("migration must append kimi, zhipu, and deepseek in the approved order")
	}

	for _, forbidden := range []string{
		"ALTER TABLE channel_monitor_v2_config",
		"DROP TABLE channel_monitor_v2_config",
		"updated_by =",
	} {
		if strings.Contains(sql, forbidden) {
			t.Fatalf("migration must preserve operator configuration; found %q", forbidden)
		}
	}
}
