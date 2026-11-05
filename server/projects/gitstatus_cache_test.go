package projects

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// stubInspectGitStatus swaps the inspection hook and counts invocations.
func stubInspectGitStatus(t *testing.T, fn func(dir string) GitStatusInfo) *int32 {
	t.Helper()
	var calls int32
	prev := inspectGitStatusFn
	inspectGitStatusFn = func(dir string) GitStatusInfo {
		atomic.AddInt32(&calls, 1)
		return fn(dir)
	}
	t.Cleanup(func() { inspectGitStatusFn = prev })
	return &calls
}

func resetGitStatusCache(t *testing.T) {
	t.Helper()
	gitStatusCache.Lock()
	gitStatusCache.entries = map[string]gitStatusCacheEntry{}
	gitStatusCache.inflight = map[string]chan struct{}{}
	gitStatusCache.Unlock()
}

func TestGetGitStatusCachesAndInvalidates(t *testing.T) {
	resetGitStatusCache(t)
	calls := stubInspectGitStatus(t, func(dir string) GitStatusInfo {
		return GitStatusInfo{IsClean: false, Commit: "abc1234", Uncommitted: 2}
	})

	first := getGitStatus("/tmp/project-a")
	if first != (GitStatusInfo{IsClean: false, Commit: "abc1234", Uncommitted: 2}) {
		t.Fatalf("unexpected status: %+v", first)
	}
	second := getGitStatus("/tmp/project-a")
	if second != first {
		t.Fatalf("cached status = %+v, want %+v", second, first)
	}
	if got := atomic.LoadInt32(calls); got != 1 {
		t.Fatalf("inspections = %d, want 1", got)
	}

	// A different project must not be served the cached value.
	getGitStatus("/tmp/project-b")
	if got := atomic.LoadInt32(calls); got != 2 {
		t.Fatalf("inspections after second project = %d, want 2", got)
	}

	InvalidateGitStatus("/tmp/project-a")
	getGitStatus("/tmp/project-a")
	if got := atomic.LoadInt32(calls); got != 3 {
		t.Fatalf("inspections after invalidate = %d, want 3", got)
	}
}

func TestGetGitStatusCoalescesConcurrentCalls(t *testing.T) {
	resetGitStatusCache(t)
	release := make(chan struct{})
	calls := stubInspectGitStatus(t, func(dir string) GitStatusInfo {
		<-release
		return GitStatusInfo{IsClean: true, Commit: "abc1234"}
	})

	const n = 8
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if got := getGitStatus("/tmp/project-c"); got.Commit != "abc1234" {
				t.Errorf("unexpected status: %+v", got)
			}
		}()
	}
	// Let every waiter reach the in-flight branch before the probe returns.
	time.Sleep(50 * time.Millisecond)
	close(release)
	wg.Wait()

	if got := atomic.LoadInt32(calls); got != 1 {
		t.Fatalf("inspections = %d, want 1 (concurrent callers must share one probe)", got)
	}
}

func TestGetGitStatusExpiresAfterTTL(t *testing.T) {
	resetGitStatusCache(t)
	calls := stubInspectGitStatus(t, func(dir string) GitStatusInfo {
		return GitStatusInfo{IsClean: true, Commit: "abc1234"}
	})

	getGitStatus("/tmp/project-d")
	gitStatusCache.Lock()
	entry := gitStatusCache.entries["/tmp/project-d"]
	entry.at = time.Now().Add(-gitStatusCacheTTL - time.Second)
	gitStatusCache.entries["/tmp/project-d"] = entry
	gitStatusCache.Unlock()

	getGitStatus("/tmp/project-d")
	if got := atomic.LoadInt32(calls); got != 2 {
		t.Fatalf("inspections after TTL = %d, want 2", got)
	}
}
