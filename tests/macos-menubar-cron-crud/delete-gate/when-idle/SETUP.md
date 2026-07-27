# Scenario

**Feature**: Delete allowed when task is idle

```
CanDeleteCronTask("idle") -> true
```

## Preconditions

Task status is `idle` (not running).

## Steps

1. Set `Status=idle`.

## Context

REQUIREMENT leaf: `delete-gate/when-idle` (scenario 3).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Status = "idle"
	return nil
}
```
