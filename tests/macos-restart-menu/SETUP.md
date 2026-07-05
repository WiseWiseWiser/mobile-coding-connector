# Scenario

**Feature**: macOS menu Restart Daemon contract and keep-alive restart APIs

```
# client: Swift sources declare menu label + endpoint
AICriticApp.swift + DaemonClient.swift -> contract assertions

# api: isolated daemon on :23312
keep-alive --kill-existing -> POST restart endpoint -> status / ping settle
```

## Preconditions

1. Repo contains `macos-ai-critic/ai-critic-macos/AICriticApp.swift` and
   `DaemonClient.swift` for client contract leaves.
2. `go build` can produce the `ai-critic` binary from the module root.
3. Keep-alive management port (`23312`) is singleton — API leaves acquire a session
   file lock to avoid parallel daemon collisions.
4. Extension startup is skipped via `AI_CRITIC_TEST_SKIP_EXTENSION=1` for fast API tests.

## Steps

1. Root `Setup` acquires keep-alive lock for API ops and sets default wait times.
2. Grouping `Setup` sets `Op` (`client` or API variant).
3. Root `Run` dispatches: read Swift contract or drive management HTTP.
4. Leaf `Assert` checks contract map or restart side effects.

## Context

Implements REQUIREMENT-DESIGN-macos-restart-daemon-menu.md. Initial RED:
`client/macos-menu-contract` fails while menu still uses `/api/keep-alive/restart`.

```go
import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func Setup(t *testing.T, req *Request) error {
	if req.StartupWaitSecs <= 0 {
		req.StartupWaitSecs = 15
	}
	if req.SettleWaitSecs <= 0 {
		req.SettleWaitSecs = 20
	}
	if req.Op == "api-restart-server" || req.Op == "api-restart-daemon" {
		unlock := acquireKeepAliveLock(t)
		t.Cleanup(unlock)
	}
	return nil
}

func acquireKeepAliveLock(t *testing.T) func() {
	session := os.Getenv("DOCTEST_SESSION_ID")
	if session == "" {
		session = fmt.Sprintf("%d", time.Now().UnixNano())
	}
	lockPath := filepath.Join(os.TempDir(), "ai-critic-restart-menu-"+session+".lock")
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		t.Skipf("another restart-menu doctest holds lock %s: %v", lockPath, err)
		return func() {}
	}
	_, _ = f.WriteString(fmt.Sprintf("%d\n", os.Getpid()))
	_ = f.Close()
	return func() { os.Remove(lockPath) }
}
```