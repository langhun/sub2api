package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

func TestRegistrationChallengesMigrationContainsOneWayControls(t *testing.T) {
	content, err := fs.ReadFile(files, "012_registration_challenges_and_budget.sql")
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	for _, fragment := range []string{
		"custom_registration_challenges",
		"custom_registration_quota_reservations",
		"custom_registration_trial_cost_reservations",
		"challenge_hash",
		"device_hash",
		"request_hash",
	} {
		if !strings.Contains(text, fragment) {
			t.Fatalf("migration missing %q", fragment)
		}
	}
}
