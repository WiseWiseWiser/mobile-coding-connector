# Scenario

**Feature**: remote-agent rejects auth import-local as local-agent-only

```
# helper name is reserved for local-agent
remote-agent auth import-local -> local-agent-only error

# credential sentinel is ignored
server-credentials(local-only-token) -> remote-agent-config.json unchanged
```

## Preconditions

The command should fail before any local credential import side effect.

## Steps

1. Set `Args` to `auth import-local`.
2. Seed isolated `~/.ai-critic/server-credentials` with `remote-must-not-import-token`.
3. Snapshot `remote-agent-config.json` before and after execution.

## Context

This leaf documents the concrete helper command name selected for the feature:
`auth import-local`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Args = []string{"auth", "import-local"}
	req.ServerCredentialContent = "remote-must-not-import-token\n"
	req.WatchRemoteConfig = true
	return nil
}
```
