package migrations

import (
	"strings"
	"testing"
)

func TestSupportTicketsMigrationCreatesModuleOwnedTables(t *testing.T) {
	content, err := files.ReadFile("002_support_tickets.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	sql := string(content)
	for _, expected := range []string{
		"CREATE TABLE IF NOT EXISTS custom_support_tickets",
		"CREATE TABLE IF NOT EXISTS custom_support_ticket_messages",
		"CREATE TABLE IF NOT EXISTS custom_support_ticket_events",
		"REFERENCES users(id)",
		"custom_support_tickets_user_activity_idx",
	} {
		if !strings.Contains(sql, expected) {
			t.Fatalf("migration missing %q", expected)
		}
	}
}

func TestSupportTicketsMigrationAcceptsLegacyWindowsChecksum(t *testing.T) {
	content, err := files.ReadFile("002_support_tickets.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	canonical, compatible := migrationChecksums(string(content))
	if want := "47e98f09d16c184f13e03f8eaec98cd02d9d65a29de333d64023a727ec89c5d3"; canonical != want {
		t.Fatalf("canonical checksum = %s, want %s", canonical, want)
	}
	legacy := "d5e3fd9f051c18bf344228151bbb265be3ade6d5806d5eb0084b2b6e0b5ced20"
	if _, ok := compatible[legacy]; !ok {
		t.Fatalf("legacy Windows checksum %s is not accepted", legacy)
	}
}
