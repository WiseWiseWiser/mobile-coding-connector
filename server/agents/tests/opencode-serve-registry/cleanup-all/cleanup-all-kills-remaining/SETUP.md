# Scenario

**Feature**: CleanupAll kills remaining children and clears registry

```
launch grok session (no stop) -> CleanupAll -> port closed, registry empty
```

## Preconditions

- Fake opencode child left running after launch.

## Steps

1. `Op = OpCleanupAll`, `SkipStop = true`.

## Context

Graceful shutdown and doctest harness must not leave headless agent ports open.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = OpCleanupAll
	req.SkipStop = true
	req.UseFakeOpenCode = true
	return nil
}
```
