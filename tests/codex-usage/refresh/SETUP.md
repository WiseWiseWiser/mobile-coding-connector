# Scenario

**Feature**: codex usage refresh skips overlapping fetches

```
slow mock + concurrent TriggerRefresh -> single exec
```

## Preconditions

`mock-slow.sh` sleeps 2s and increments counter file.

## Steps

1. Set `Op=refresh` in leaf.

## Context

Validates skip-concurrent-fetch requirement.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "refresh"
	return nil
}
```