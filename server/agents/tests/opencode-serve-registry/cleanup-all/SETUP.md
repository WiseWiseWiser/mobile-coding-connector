# Scenario

**Feature**: CleanupAllOpencodeServe shutdown hook

```
launch without stop -> TestExported_CleanupAllOpencodeServe -> all children dead
```

## Preconditions

- Launched headless session left running intentionally.

## Steps

1. Set `Op = OpCleanupAll` in leaf.

## Context

Mirrors `agents.Shutdown()` extended to kill registered children.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.UseFakeOpenCode = true
	return nil
}
```
