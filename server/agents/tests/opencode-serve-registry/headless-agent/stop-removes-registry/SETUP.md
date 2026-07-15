# Scenario

**Feature**: stop removes registry entry and closes port

```
launch -> registry entry -> TestExported_StopAgentSession -> entry gone, port closed
```

## Preconditions

- Fake opencode on PATH.

## Steps

1. `Op = OpStopRegistry`.

## Context

Path A stop hook must unregister after verified kill.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpStopRegistry
	req.UseFakeOpenCode = true
	return nil
}
```
