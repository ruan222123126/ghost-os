package orchestration

import (
	"testing"

	bridgeconfig "ghost-os/bridge/config"
)

func TestConfigResponseFromSnapshotIncludesWebRooterFlags(t *testing.T) {
	response := configResponseFromSnapshot(bridgeconfig.Snapshot{
		WebRooterEnabled:     true,
		WebRooterAPITokenSet: true,
	})

	if !response.WebRooterEnabled {
		t.Fatal("expected web_rooter_enabled to be true")
	}
	if !response.WebRooterAPITokenSet {
		t.Fatal("expected web_rooter_api_token_set to be true")
	}
}
