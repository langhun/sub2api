package migrations

import (
	"strings"
	"testing"
)

func TestChannelMonitorV2ActiveProbesMigrationIsAppendOnlyAndIdempotent(t *testing.T) {
	content, err := files.ReadFile("017_channel_monitor_v2_active_probes.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	sql := string(content)
	for _, expected := range []string{
		"CREATE TABLE IF NOT EXISTS custom_channel_monitor_v2_active_probe_results",
		"REFERENCES channel_monitors(id) ON DELETE CASCADE",
		"UNIQUE (monitor_id, model, checked_at)",
		"CREATE INDEX IF NOT EXISTS idx_custom_channel_monitor_v2_active_probe_lookup",
		"interval_seconds BETWEEN 15 AND 3600",
	} {
		if !strings.Contains(sql, expected) {
			t.Fatalf("migration missing %q", expected)
		}
	}
	for _, forbidden := range []string{"ALTER TABLE channel_monitor_", "UPDATE channel_monitor_", "DROP TABLE"} {
		if strings.Contains(sql, forbidden) {
			t.Fatalf("migration must not rewrite upstream schema or history; found %q", forbidden)
		}
	}
}
