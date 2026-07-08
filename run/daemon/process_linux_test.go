//go:build linux

package daemon

import (
	"os"
	"testing"
)

func TestIsProcessStopped(t *testing.T) {
	if IsProcessStopped(0) {
		t.Fatal("pid 0 should not be stopped")
	}
	if IsProcessStopped(os.Getpid()) {
		t.Fatal("current process should not report stopped")
	}
	if IsProcessStopped(99999999) {
		t.Fatal("missing pid should not report stopped")
	}
}