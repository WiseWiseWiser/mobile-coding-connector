# Scenario

**Feature**: graceful server shutdown cleans agent children

```
launch grok -> SIGTERM ai-critic-server -> CleanupAll -> port closed
```

## Preconditions

- Session left running when server receives SIGTERM.

## Steps

1. `Scenario = ScenarioShutdown`.

## Context

Mirrors production `agents.Shutdown()` extended with CleanupAllOpencodeServe.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Scenario = ScenarioShutdown
	return nil
}
```
