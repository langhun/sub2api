package registrationprotection

import "testing"

func TestMaskRiskEmailDoesNotExposeFullLocalPart(t *testing.T) {
	if got := maskRiskEmail("alice@example.com"); got != "a***e@example.com" {
		t.Fatalf("maskRiskEmail() = %q", got)
	}
	if got := maskRiskEmail("a@example.com"); got != "*@example.com" {
		t.Fatalf("maskRiskEmail() short local = %q", got)
	}
}
