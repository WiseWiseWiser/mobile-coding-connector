# Scenario

**Feature**: local-agent imports local server credentials into local config

```
# local-only helper reads ~/.ai-critic/server-credentials
local-agent auth import-local -> first non-empty credential line

# helper writes local-agent-config.json without exposing the token
credential token -> local-agent-config.json domains/default
```

## Preconditions

The command is local-profile-only. The credential source is the local server
credential file under the isolated user home.

## Steps

1. Child leaves seed credential-file content and any pre-existing local config.
2. Run `local-agent auth import-local`.
3. Snapshot `local-agent-config.json` after the command.

## Context

Split factor: local credential import command, distinct from normal API-call auth failures.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Args = []string{"auth", "import-local"}
	req.WatchLocalConfig = true
	return nil
}
```
