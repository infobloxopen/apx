package integration

import (
	"os"
	"testing"
)

// TestConsumerWorkflow validates the full consumer experience
// following quickstart.md: search, add, gen, sync, unlink
func TestConsumerWorkflow(t *testing.T) {
	if os.Getenv("CI") == "" && os.Getenv("GITEA_URL") == "" {
		t.Skip("Skipping integration test (set CI=1 or GITEA_URL to run)")
	}

	// TODO: Implement full consumer workflow test after Phase 5 implementation
	// This test will validate:
	// 1. Search APIs in catalog
	// 2. Add dependencies to apx.lock
	// 3. Generate code with overlays
	// 4. Sync go.work entries
	// 5. Unlink and switch to published modules
	// 6. Verify import paths remain stable throughout

	t.Skip("Consumer workflow test pending implementation - see T061")
}
