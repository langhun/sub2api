package registrationprotection

import "testing"

func TestEvidenceForSignalsUsesStableDetails(t *testing.T) {
	got := evidenceForSignals([]string{"exact_fingerprint_repeat", "api_key_created", "future_signal"})
	if len(got) != 3 {
		t.Fatalf("expected 3 evidence entries, got %d", len(got))
	}
	if got[0].Signal != "exact_fingerprint_repeat" || got[0].Detail != "fingerprint_repeat" {
		t.Fatalf("unexpected fingerprint evidence: %#v", got[0])
	}
	if got[1].Detail != "api_key_created" {
		t.Fatalf("unexpected API key evidence: %#v", got[1])
	}
	if got[2].Detail != "unknown" {
		t.Fatalf("unknown signals must remain explainable: %#v", got[2])
	}
}

func TestEvidenceForSignalsReturnsEmptySlice(t *testing.T) {
	got := evidenceForSignals(nil)
	if got == nil || len(got) != 0 {
		t.Fatalf("expected a non-nil empty evidence list, got %#v", got)
	}
}
