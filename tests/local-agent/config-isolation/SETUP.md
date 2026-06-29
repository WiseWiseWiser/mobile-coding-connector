# Scenario

**Feature**: local-agent uses only its own config file

```
# remote-agent-config.json present as sentinel; local-agent reads local-agent-config.json only
local-agent -> loadConfig(local path) -> (must not mutate remote file)
```

## Preconditions

Both config files may exist under the same `~/.ai-critic` directory.

## Steps

1. Child leaves snapshot `remote-agent-config.json` around a command.

## Context

Q2 decision: separate config file per CLI binary.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	if req.WatchRemoteConfig && req.SeedRemoteConfig == nil {
		t.Logf("config-isolation: WatchRemoteConfig without SeedRemoteConfig in %s", t.Name())
	}
	return nil
}
```