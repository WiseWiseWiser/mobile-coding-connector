package tools

import (
	"testing"
	"time"
)

func TestSnapshotInstalledToolsBoundedTime(t *testing.T) {
	start := time.Now()
	snaps := SnapshotInstalledTools()
	elapsed := time.Since(start)

	if elapsed > snapshotTotalTimeout+2*time.Second {
		t.Fatalf("snapshot took %v, want <= %v", elapsed, snapshotTotalTimeout)
	}

	for _, snap := range snaps {
		if snap.Name == "" {
			t.Fatalf("snapshot entry missing name: %+v", snap)
		}
		if snap.Path == "" {
			t.Fatalf("snapshot entry missing path: %+v", snap)
		}
	}
}