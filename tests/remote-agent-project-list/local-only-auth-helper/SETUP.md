# Scenario

**Feature**: local credential import helper is not available in remote-agent

```
# remote profile receives local-only helper command
remote-agent auth import-local -> reject before credential import

# local credential sentinel must not be read into remote config or output
~/.ai-critic/server-credentials -> (not used by remote-agent)
```

## Preconditions

The isolated home may contain a local server credential file, but remote-agent must not
use it for config bootstrapping.

## Steps

1. Child leaves choose the local-only helper command.
2. Snapshot `remote-agent-config.json` before and after the command.

## Context

Split factor: local-only command rejection for the remote profile.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.WatchRemoteConfig = true
	return nil
}
```
