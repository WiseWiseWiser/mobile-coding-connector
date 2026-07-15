# Scenario

**Feature**: opencode-serve-children registry lifecycle harness

```
launch -> write opencode-serve-children.json -> stop/CleanupAll -> remove entry
```

## Preconditions

- Isolated `AI_CRITIC_HOME` per leaf via `lib.CreateTestConfigHome()`.
- Fake opencode on PATH unless leaf requests real opencode (`UseRealOpenCode`).
- `t.Cleanup` uses `lib.CleanupOpencodeServe` + stop exports (no `pkill -f`).
- Implementer adds registry persistence and `TestExported_*` helpers listed in root DOCTEST.md.

## Steps

1. Child `Setup` sets `Request.Op` and scenario flags.
2. Root `Run` launches/stops agents and reads registry via exports.
3. Leaf `Assert` checks registry JSON, port state, cleanup outcome.

## Context

Classic TDD: all leaves RED until `opencode_serve_children/registry.go` and launch/stop hooks exist.

```go
import (
	"os/exec"
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	if _, err := exec.LookPath("lsof"); err != nil {
		t.Skip("lsof required for registry port checks")
	}
	return nil
}
```
