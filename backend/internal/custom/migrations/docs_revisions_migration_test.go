package migrations

import (
	"strings"
	"testing"
)

func TestDocsRevisionsMigrationCreatesImmutableRevisionLedger(t *testing.T) {
	content, err := files.ReadFile("007_docs_revisions.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	sql := string(content)
	for _, expected := range []string{
		"CREATE TABLE IF NOT EXISTS custom_docs_revisions",
		"author_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE",
		"custom_docs_revisions_lookup_idx",
		"CHECK (locale IN ('zh', 'en'))",
	} {
		if !strings.Contains(sql, expected) {
			t.Fatalf("migration missing %q", expected)
		}
	}
	if strings.Contains(sql, "DROP TABLE") || strings.Contains(sql, "ALTER TABLE users") {
		t.Fatal("docs migration must not alter upstream tables")
	}
}
