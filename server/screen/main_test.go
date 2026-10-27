package screen

import (
	"os"
	"testing"
)

// The tests must not depend on whether the machine running them happens to be
// locked, so the lock probe is pinned to "unlocked" by default.
func TestMain(m *testing.M) {
	lockProbe = func() bool { return false }
	// Never reap real ffmpeg processes from a unit test run.
	reapStale = func() {}
	os.Exit(m.Run())
}
