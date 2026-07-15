# Scenario

**Feature**: wrkserver HTTP list/create/register against isolated WRK_HOME

```
# host mounts wrkserver; tests call handlers or Register(mux, base)
wrkserver.New(Options{WrkHome}) -> ListProjects | CreateWorktree | Register
  -> httptest Response (JSON projects / path+branch / error)
```

## Preconditions

1. Package `github.com/xhd2015/wrk/wrkcli/wrkserver` exports
   `Options`, `New`, `ListProjects`, `CreateWorktree`, and `Register`.
2. `git` is available in PATH for leaves that seed real repositories.
3. Each leaf uses an isolated temp `WrkHome` (never the developer's `~/.wrk`).
4. No dependency on an installed `wrk` binary.

## Steps

1. Root `Setup` allocates a temp `WrkHome` on every leaf.
2. Grouping `Setup` sets `Op` (`list` / `create` / `register`).
3. Leaf `Setup` seeds `projects.json`, git repos, create body fields, or
   Register base/path as needed.
4. Root `Run` constructs `wrkserver.New` and dispatches via httptest or mux.
5. Leaf `Assert` checks status codes and JSON fields.

## Context

Implements REQUIREMENT-DESIGN-wrkserver-projects-menubar.md section A (handlers
+ Register). Module path is `github.com/xhd2015/wrk/wrkcli/wrkserver`.

```go
import (
	"os"
	"os/exec"
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available in PATH")
	}
	if req.WrkHome == "" {
		req.WrkHome = mkTempDir(t, "wrkserver-home-*")
	}
	return nil
}

// ensureWrkHome is a no-op guard used by leaves that need an empty registry dir.
func ensureWrkHome(t *testing.T, req *Request) {
	t.Helper()
	if req.WrkHome == "" {
		req.WrkHome = mkTempDir(t, "wrkserver-home-*")
	}
	if err := os.MkdirAll(req.WrkHome, 0o755); err != nil {
		t.Fatalf("mkdir WrkHome: %v", err)
	}
}
```
