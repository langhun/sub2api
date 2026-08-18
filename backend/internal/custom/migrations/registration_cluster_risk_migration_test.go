package migrations

import (
	"strings"
	"testing"
)

func TestRegistrationClusterRiskMigrationIsAdditiveAndHasAuditState(t *testing.T) {
	content, err := files.ReadFile("010_registration_cluster_risk.sql")
	if err != nil {
		t.Fatalf("read cluster risk migration: %v", err)
	}
	sql := string(content)
	for _, expected := range []string{
		"custom_registration_cluster_events",
		"custom_registration_risk_profiles",
		"custom_registration_risk_audit",
		"network_hash TEXT NOT NULL",
		"user_agent_hash TEXT NOT NULL",
		"payment_unlocked BOOLEAN",
		"override_reason TEXT",
	} {
		if !strings.Contains(sql, expected) {
			t.Fatalf("cluster risk migration missing %q", expected)
		}
	}
	if strings.Contains(sql, "DROP TABLE") || strings.Contains(sql, "TRUNCATE") {
		t.Fatal("cluster risk migration must be additive")
	}
}
