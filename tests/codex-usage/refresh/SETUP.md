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
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "refresh"
	return nil
}
```