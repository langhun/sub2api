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

func TestMigrationChecksumsTreatLineEndingsAsEquivalent(t *testing.T) {
	lf := "CREATE TABLE example (id BIGINT);\nSELECT 1;"
	crlf := strings.ReplaceAll(lf, "\n", "\r\n")
	canonicalLF, compatibleLF := migrationChecksums(lf)
	canonicalCRLF, compatibleCRLF := migrationChecksums(crlf)

	if canonicalLF != canonicalCRLF {
		t.Fatalf("canonical checksums differ: LF=%s CRLF=%s", canonicalLF, canonicalCRLF)
	}
	for _, checksum := range []string{migrationChecksum(lf), migrationChecksum(crlf)} {
		if _, ok := compatibleLF[checksum]; !ok {
			t.Fatalf("LF compatibility set missing %s", checksum)
		}
		if _, ok := compatibleCRLF[checksum]; !ok {
			t.Fatalf("CRLF compatibility set missing %s", checksum)
		}
	}
	if mutated, _ := migrationChecksums(lf + "\nSELECT 2;"); mutated == canonicalLF {
		t.Fatal("semantic migration change must alter the checksum")
	}
}

func TestSmartRoutingMigrationAcceptsPreviouslyAppliedChecksum(t *testing.T) {
	content, err := files.ReadFile("012_smart_routing.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	_, compatible := migrationChecksums(strings.TrimSpace(string(content)))
	for _, checksum := range historicalMigrationChecksums["012_smart_routing.sql"] {
		if _, ok := compatible[checksum]; ok {
			t.Fatalf("historical checksum must be explicitly accepted by Apply, not current content: %s", checksum)
		}
	}
	const deployed = "f8788dd4063924c194ee09180cd6d9073ba22c02e149ecd3c57a12c99bf6627e"
	found := false
	for _, checksum := range historicalMigrationChecksums["012_smart_routing.sql"] {
		if checksum == deployed {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing deployed smart-routing checksum compatibility entry")
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

func TestCheckinUserAccessMigrationStaysInCustomSchema(t *testing.T) {
	content, err := files.ReadFile("014_checkin_user_access.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	sql := string(content)
	for _, expected := range []string{
		"CREATE TABLE IF NOT EXISTS custom_checkin_user_access",
		"REFERENCES users(id) ON DELETE CASCADE",
		"updated_by BIGINT REFERENCES users(id) ON DELETE SET NULL",
	} {
		if !strings.Contains(sql, expected) {
			t.Fatalf("migration missing %q", expected)
		}
	}
	if strings.Contains(sql, "ALTER TABLE users") || strings.Contains(sql, "DROP TABLE users") {
		t.Fatal("custom migration must not modify the upstream users table")
	}
}

func TestSmartRoutingOutcomeMigrationStoresFinalRequestResult(t *testing.T) {
	content, err := files.ReadFile("018_smart_routing_outcomes.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	sql := string(content)
	for _, expected := range []string{
		"CREATE TABLE IF NOT EXISTS custom_route_outcomes",
		"request_id TEXT PRIMARY KEY",
		"final_success BOOLEAN NOT NULL",
		"group_switches INTEGER NOT NULL",
		"custom_route_outcomes_key_created_idx",
		"custom_route_outcomes_created_idx",
	} {
		if !strings.Contains(sql, expected) {
			t.Fatalf("migration missing %q", expected)
		}
	}
}

func TestSmartRoutingExplorationMigrationCapsHistoricalProfiles(t *testing.T) {
	content, err := files.ReadFile("019_smart_routing_exploration_cap.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	sql := string(content)
	for _, expected := range []string{
		"WHEN exploration_rate < 0 THEN 0",
		"WHERE exploration_rate < 0 OR exploration_rate > 0.01",
		"CHECK (exploration_rate >= 0 AND exploration_rate <= 0.01)",
		"WHEN insufficient_privilege THEN",
		"skipping owner-only smart routing constraint hardening",
	} {
		if !strings.Contains(sql, expected) {
			t.Fatalf("migration missing %q", expected)
		}
	}
}
