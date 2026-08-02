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
