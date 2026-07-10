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
	"testing"
	"time"
)

func Setup(t *testing.T, req *Request) error {
	req.Op = "auth"
	req.Dir = t.TempDir()
	req.OmitMode = true
	req.OmitSend = true

	// Serialize auth leaves: auth.SetCredentialsFile is process-global.
	unlock := acquireLocalITerm2AuthLock(t)
	t.Cleanup(unlock)

	credDir := t.TempDir()
	credPath := filepath.Join(credDir, "server-credentials")
	if err := os.WriteFile(credPath, []byte("test-iterm2-token\n"), 0o600); err != nil {
		t.Fatalf("write credentials: %v", err)
	}
	req.CredentialsPath = credPath
	return nil
}

func acquireLocalITerm2AuthLock(t *testing.T) func() {
	session := DOCTEST_SESSION_ID
	if session == "" {
		session = fmt.Sprintf("%d", time.Now().UnixNano())
	}
	lockPath := filepath.Join(os.TempDir(), "local-iterm2-open-auth-"+session+".lock")
	deadline := time.Now().Add(30 * time.Second)
	for {
		f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err == nil {
			_, _ = f.WriteString(fmt.Sprintf("%d\n", os.Getpid()))
			_ = f.Close()
			return func() { os.Remove(lockPath) }
		}
		if time.Now().After(deadline) {
			t.Skipf("could not acquire auth lock %s: %v", lockPath, err)
			return func() {}
		}
		time.Sleep(50 * time.Millisecond)
	}
}
```
