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

func TestDocsRevisionReviewMigrationAddsApprovalState(t *testing.T) {
	content, err := files.ReadFile("008_docs_revision_review.sql")
	if err != nil {
		t.Fatalf("read review migration: %v", err)
	}
	sql := string(content)
	for _, expected := range []string{
		"ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'approved'",
		"reviewed_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL",
		"CHECK (status IN ('pending', 'approved', 'rejected'))",
		"custom_docs_revisions_review_idx",
	} {
		if !strings.Contains(sql, expected) {
			t.Fatalf("review migration missing %q", expected)
		}
	}
}
