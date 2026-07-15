# Scenario

**Feature**: no orphan after graceful server shutdown

```
launch grok -> SIGTERM server (no DELETE) -> child port closed, registry cleared
```

## Preconditions

- Server graceful shutdown calls agents.Shutdown → CleanupAll.

## Steps

1. Inherited `ScenarioShutdown`.

## Context

Primary production fix: abrupt test end no longer leaves launchd orphans.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	if req.AgentID == "" {
		req.AgentID = "grok"
	}
	return nil
}
```
