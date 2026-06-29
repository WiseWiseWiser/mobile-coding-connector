# Scenario

**Feature**: local-agent CLI integration harness

```
# build binaries, isolated HOME, optional server, run local-agent with test hook env
test harness -> local-agent subprocess -> ai-critic-server (optional)
testhooks env -> override default port / reachability in child
```

## Preconditions

1. Module builds `ai-critic-server` (`.`) and `local-agent` (`./cmd/local-agent`).
2. `cmd/agentcli/testhooks` supplies env helpers consumed at `local-agent` startup.
3. Each test uses a fresh temp `HOME` with `~/.ai-critic/` for agent config files.

## Steps

1. Root `Run` builds server and `local-agent` binaries into the temp dir.
2. Creates isolated `HOME` and optional `local-agent-config.json` / sentinel `remote-agent-config.json`.
3. Starts `ai-critic-server` when `Request.StartServer` is true; waits for `/ping`.
4. Runs `local-agent` with `Request` flags and subcommand args; applies test hook env vars.
5. Captures stdout, stderr, exit code; optional before/after remote config snapshot.

## Context

Implements REQUIREMENT-DESIGN-local-agent-cli.md. Complements future `cmd/agentcli` unit
tests; this tree proves profile-specific behavior through the real binary.

```go
import (
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	if req.Args == nil {
		req.Args = []string{}
	}
	return nil
}
```