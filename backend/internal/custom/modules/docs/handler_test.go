package docs

import "testing"

func TestNormalizeLocalePreservesInvalidValuesForValidation(t *testing.T) {
	for value, expected := range map[string]string{
		"":   "zh",
		"zh": "zh",
		"en": "en",
		"fr": "fr",
	} {
		if got := normalizeLocale(value); got != expected {
			t.Fatalf("normalizeLocale(%q) = %q, want %q", value, got, expected)
		}
	}
}
