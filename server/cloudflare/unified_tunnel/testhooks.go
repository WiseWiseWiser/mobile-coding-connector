package unified_tunnel

import "sync/atomic"

var (
	testStartProcessHook         func(*UnifiedTunnelManager) error
	testStopProcessHook          func(*UnifiedTunnelManager)
	testPostRestartSideEffectsOff bool
	testRebuildExecutedCount     atomic.Int32
)

func getTestStartProcessHook() func(*UnifiedTunnelManager) error {
	return testStartProcessHook
}

func getTestStopProcessHook() func(*UnifiedTunnelManager) {
	return testStopProcessHook
}

func postRestartSideEffectsDisabled() bool {
	return testPostRestartSideEffectsOff
}

func recordRebuildExecutedForTest() {
	if testStartProcessHook != nil || testStopProcessHook != nil || testPostRestartSideEffectsOff {
		testRebuildExecutedCount.Add(1)
	}
}

// SetTestProcessHooks installs hooks that bypass real cloudflared process management.
// Returns a cleanup function that restores defaults.
func SetTestProcessHooks(
	start func(*UnifiedTunnelManager) error,
	stop func(*UnifiedTunnelManager),
) func() {
	testStartProcessHook = start
	testStopProcessHook = stop
	testPostRestartSideEffectsOff = true
	testRebuildExecutedCount.Store(0)
	return func() {
		testStartProcessHook = nil
		testStopProcessHook = nil
		testPostRestartSideEffectsOff = false
		testRebuildExecutedCount.Store(0)
	}
}

// TestRebuildExecutedCount returns how many rebuild/restart cycles ran while test hooks were active.
func TestRebuildExecutedCount() int {
	return int(testRebuildExecutedCount.Load())
}