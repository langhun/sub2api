package registrationprotection

import "testing"

func TestNormalizeNetworkIPv4Uses24(t *testing.T) {
	got, ok := normalizeNetwork("203.0.113.37")
	if !ok || got != "203.0.113.0/24" {
		t.Fatalf("normalizeNetwork() = %q, %v", got, ok)
	}
}

func TestNormalizeNetworkIPv6Uses64(t *testing.T) {
	got, ok := normalizeNetwork("2001:db8:abcd:1234:1:2:3:4")
	if !ok || got != "2001:db8:abcd:1234::/64" {
		t.Fatalf("normalizeNetwork() = %q, %v", got, ok)
	}
}

func TestHashValueDoesNotExposeInput(t *testing.T) {
	const input = "203.0.113.0/24"
	if got := hashValue(input); got == input || len(got) != 64 {
		t.Fatalf("hashValue() = %q", got)
	}
}
