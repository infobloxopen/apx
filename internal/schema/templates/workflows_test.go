package templates

import (
	"strings"
	"testing"
)

func TestGenerateCanonicalOnMerge_LowercaseImage(t *testing.T) {
	out := GenerateCanonicalOnMerge("Infoblox-CTO")

	// The GHCR image reference must be lowercase regardless of org casing.
	if !strings.Contains(out, "ghcr.io/infoblox-cto/") {
		t.Errorf("expected lowercase org in IMAGE, got:\n%s", out)
	}
	if strings.Contains(out, "ghcr.io/Infoblox-CTO/") {
		t.Errorf("IMAGE contains mixed-case org, which will fail docker push")
	}
}

func TestGenerateCatalogDockerfile_PreservesCasing(t *testing.T) {
	out := GenerateCatalogDockerfile("Infoblox-CTO")

	// The vendor label is metadata — it should preserve the original casing.
	if !strings.Contains(out, `"Infoblox-CTO"`) {
		t.Errorf("expected original casing in vendor label, got:\n%s", out)
	}
}
