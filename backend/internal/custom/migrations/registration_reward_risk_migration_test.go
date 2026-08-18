package migrations

import (
	"strings"
	"testing"
)

func TestRegistrationRewardRiskMigrationCreatesImmutableFingerprintLedger(t *testing.T) {
	content, err := files.ReadFile("009_registration_reward_risk.sql")
	if err != nil {
		t.Fatalf("read registration risk migration: %v", err)
	}
	sql := string(content)
	for _, expected := range []string{
		"CREATE TABLE IF NOT EXISTS custom_registration_fingerprint_history",
		"CREATE TABLE IF NOT EXISTS custom_registration_fingerprint_events",
		"CREATE TABLE IF NOT EXISTS custom_registration_fingerprint_reservations",
		"REFERENCES users(id) ON DELETE SET NULL",
		"custom_registration_fingerprint_events_user_idx",
		"custom_registration_fingerprint_reservations_expiry_idx",
		"CHECK (length(fingerprint_hash) = 64)",
	} {
		if !strings.Contains(sql, expected) {
			t.Fatalf("registration risk migration missing %q", expected)
		}
	}
	if strings.Contains(sql, "ALTER TABLE users") || strings.Contains(sql, "DROP TABLE") {
		t.Fatal("registration risk migration must not alter or drop upstream tables")
	}
}
