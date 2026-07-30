# Scenario

**Feature**: remote-agent `service add` / `service rm` / `list --all` L2 harness

```
# L2: services.NewManagerAt + RegisterAPIWithManager + agentcli.Run
leaf Setup -> optional services.json seed -> httptest mux (Bearer)
       -> agentcli.Run(--server, --token, service …)
       -> ListAll + services.json snapshot
```

## Preconditions

1. Classic TDD: product CLI `service add` / `service rm` / `list --all` may be
   missing → runtime RED until implementer lands client + agentcli wiring.
2. Server Manager already supports POST create, DELETE by id, and `GET ?all=1`.
3. Each leaf uses isolated `lib.CreateTestConfigHome` (no process-global env).
4. CLI leaves pass `--server` + `--token` (`lib.TestPassword`); agentcli.Run is
   mutex-serialized (stdout/stderr capture).
5. Long-running commands use `sleep` so PID checks stay stable.

## Steps

1. Root `Run` creates config home, seeds `services.json` when requested, mounts
   `RegisterAPIWithManager` on httptest with bearer auth.
2. Leaf `Setup` sets `CLIArgs` (+ optional seeds / target name).
3. `Run` executes `agentcli.Run` and snapshots `ListAll` + on-disk JSON.
4. Leaf `Assert` checks exit code, stdout templates, and persistence.

## Context

Implements `/tmp/REQUIREMENT-DESIGN-remote-agent-service-add-rm.md`.
Mirror of `tests/managed-service-enable-disable` CLI L2 path. L2 only; no e2e label.

```go
import (
	"testing"

	"github.com/xhd2015/ai-critic/script/lib"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	if req.Token == "" {
		req.Token = lib.TestPassword
	}
	return nil
}

// setCLI configures a CLI leaf (leading "service" required).
func setCLI(req *Request, args ...string) {
	req.CLIArgs = args
}
```
