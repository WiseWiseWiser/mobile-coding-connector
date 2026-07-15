# Scenario

**Feature**: shared opencode serve cleanup helper harness

```
# read registry + lsof ports -> CollectOpencodeServePIDs
opencode-serve-children.json + lsof tcp:PORT -> Collect -> []pid

# verify ps command -> SIGTERM/SIGKILL -> clear registry
Collect -> KillOpencodeServePIDs (ps verify) -> CleanupOpencodeServe -> empty JSON
```

## Preconditions

- Implementer adds `script/lib/opencode_cleanup.go` with:
  - `CollectOpencodeServePIDs(configHome string, extraPorts ...int) ([]int, error)`
  - `KillOpencodeServePIDs(configHome string, pids []int) (skipped []int, killed []int, err error)`
  - `CleanupOpencodeServe(configHome string, extraPorts ...int) error`
- Each leaf uses isolated `AI_CRITIC_HOME` via `lib.CreateTestConfigHome()`.
- Fake opencode built from `server/agents/tests/grok-integration/testdata/fake-opencode`.
- No `pkill -f` in any test path.

## Steps

1. Child `Setup` sets `Request.Op` and scenario flags.
2. Root `Run` prepares config home, fixtures, subprocesses, invokes lib helpers.
3. Leaf `Assert` checks PIDs, port state, registry contents.

## Context

Classic TDD: leaves are RED until `opencode_cleanup.go` and registry persistence land.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	return nil
}
```
