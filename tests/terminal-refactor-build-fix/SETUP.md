# Scenario

**Bug**: post-terminal-refactor builds break until ai-critic wires the dot-pkgs core

```
# compile regression
go build ./cmd/remote-agent -> must succeed (no missing WS helpers in exec.go)
go build ./ -> must succeed (ShellQuote from dot-pkgs ptywrap, not server/terminal)

# shell quoting
run/keep_alive.go -> dot-pkgs ptywrap.ShellQuote -> POSIX-safe script embedding

# interactive exec client
remote-agent exec (TTY) -> dot-pkgs ptywrap/client -> /api/exec/ws fake server
```

## Preconditions

1. ai-critic module root resolvable via `DOCTEST_ROOT` or `go.mod` walk.
2. `sh` available in PATH for shell-quote and keep-alive syntax checks.
3. Implementer adds `run.TestExported_OutputKeepAliveScript` for keep-alive leaf
   (accepts explicit `binPath` with spaces).

## Steps

1. Root `Run` dispatches on `req.Phase` via `buildfixtest` harness.
2. Leaf `Setup` sets `Phase` and scenario-specific fields on `Request`.

## Context

Implements REQUIREMENT-DESIGN-terminal-refactor-build-fix.md. Tests are RED until
ai-critic call sites import the dot-pkgs core directly.

```go
import (
	"testing"

	"github.com/xhd2015/ai-critic/tests/terminal-refactor-build-fix/testdata/buildfixtest"
)

func Setup(t *testing.T, req *Request) error {
	_ = buildfixtest.ModuleRoot(t)
	return nil
}
```