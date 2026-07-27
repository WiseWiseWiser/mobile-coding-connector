# Scenario

**Feature**: keep-alive `--kill-existing` port conflict resolution

```
# optional port occupiers bind server/daemon ports
port occupier(s) -> keep-alive [--kill-existing] -> kill occupiers -> daemon binds -> status API
```

## Preconditions

1. Module builds `ai-critic` and `testdata/port-occupier` helper.
2. Isolated `AI_CRITIC_HOME` with test credentials.
3. Keep-alive management port (`23312`) is singleton — session file lock prevents
   parallel daemon collisions.
4. Server port occupier serves `GET /ping` → `pong` when simulating managed server conflict.

## Steps

1. Root `Setup` acquires keep-alive session lock and assigns default `ServerPort`.
2. Leaf `Setup` sets `KillExisting`, occupier flags, and expectations.
3. Root `Run` starts occupiers, launches keep-alive, polls status or exit error.
4. Leaf `Assert` verifies occupier PIDs, daemon start, and API response.

## Context

Implements REQUIREMENT-DESIGN-macos-app-and-bar.md Feature 1. Complements
`tests/keep-alive/` (bootstrap timing) without modifying that tree.

```go
import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/xhd2015/ai-critic/server/config"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	unlock := acquireKeepAliveLock(t, d)
	t.Cleanup(unlock)

	if req.ServerPort <= 0 {
		req.ServerPort = config.DefaultServerPort
	}
	if req.StartupWaitSecs <= 0 {
		req.StartupWaitSecs = 15
	}
	return nil
}

func acquireKeepAliveLock(t *testing.T, d *session.Doctest) func() {
	// Share the keep-alive global flock with tests/keep-alive (port 23312).
	_ = d
	lockPath := filepath.Join(os.TempDir(), "ai-critic-keepalive-doctest-global.lock")
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		t.Fatalf("open keep-alive lock %s: %v", lockPath, err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		_ = f.Close()
		t.Fatalf("flock keep-alive lock %s: %v", lockPath, err)
	}
	_, _ = f.WriteAt([]byte(fmt.Sprintf("%d\n", os.Getpid())), 0)
	return func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
	}
}
```