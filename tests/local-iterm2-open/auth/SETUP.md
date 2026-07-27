# Scenario

**Feature**: Bearer auth required for open endpoint

```
auth.Middleware(mux) + credentials file
  no Authorization -> 401
  Bearer valid-token -> 200
```

## Preconditions

1. Temp credentials file with a known token.
2. `auth.SetCredentialsFile` points at that file for the leaf.
3. Middleware skip list empty (forces auth on all `/api/*`).

## Steps

1. Set `Op=auth`.
2. Write credentials; leaf sets Bearer or omits it.

## Context

REQUIREMENT scenario 6.

```go
import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	req.Op = "auth"
	req.Dir = t.TempDir()
	req.OmitMode = true
	req.OmitSend = true

	// Serialize auth leaves: auth.SetCredentialsFile is process-global.
	unlock := acquireLocalITerm2AuthLock(t, d)
	t.Cleanup(unlock)

	credDir := t.TempDir()
	credPath := filepath.Join(credDir, "server-credentials")
	if err := os.WriteFile(credPath, []byte("test-iterm2-token\n"), 0o600); err != nil {
		t.Fatalf("write credentials: %v", err)
	}
	req.CredentialsPath = credPath
	return nil
}

func acquireLocalITerm2AuthLock(t *testing.T, d *session.Doctest) func() {
	sid := ""
	if d != nil {
		sid = d.DOCTEST_SESSION_ID
	}
	if sid == "" {
		sid = "default"
	}
	// Global flock: credentials path is process-global across parallel leaves.
	_ = sid
	lockPath := filepath.Join(os.TempDir(), "local-iterm2-open-auth-global.lock")
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		t.Fatalf("open auth lock %s: %v", lockPath, err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		_ = f.Close()
		t.Fatalf("flock auth lock %s: %v", lockPath, err)
	}
	_, _ = f.WriteAt([]byte(fmt.Sprintf("%d\n", os.Getpid())), 0)
	return func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
	}
}
```
