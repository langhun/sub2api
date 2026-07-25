package walletextension

import "testing"

func TestMetadataDeclaresStableContractBoundary(t *testing.T) {
	if Metadata.ID != ModuleID {
		t.Fatalf("metadata ID = %q, want %q", Metadata.ID, ModuleID)
	}
	if Metadata.Version != ModuleVersion {
		t.Fatalf("metadata version = %q, want %q", Metadata.Version, ModuleVersion)
	}
	if Metadata.Status != StatusOperational {
		t.Fatalf("metadata status = %q, want %q", Metadata.Status, StatusOperational)
	}

	seen := make(map[Dependency]struct{}, len(Metadata.Dependencies))
	for _, dependency := range Metadata.Dependencies {
		if dependency == "" {
			t.Fatal("metadata includes an empty dependency")
		}
		if _, exists := seen[dependency]; exists {
			t.Fatalf("metadata declares duplicate dependency %q", dependency)
		}
		seen[dependency] = struct{}{}
	}
}
