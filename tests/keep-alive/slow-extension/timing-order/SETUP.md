# Scenario

**Feature**: core bootstrap completes before extension tasks run

```
# shorter 5s extension delay still proves ordering via bootstrap markers
server core_ready -> /ping -> extension_start / auto-task logs
```

## Preconditions

Slow extension path with moderate delay for log separation.

## Steps

1. Default `ExtensionDelayMs` to 5000 when unset.

## Context

Ordering assertions use `[bootstrap]` phases or legacy auto-task lines until
markers land.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	if req.ExtensionDelayMs <= 0 {
		req.ExtensionDelayMs = 5000
	}
	if req.ObserveSecs <= 0 {
		req.ObserveSecs = 10
	}
	return nil
}
```