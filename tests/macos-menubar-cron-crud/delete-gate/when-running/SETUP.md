# Scenario

**Feature**: Delete disabled while task is running

```
CanDeleteCronTask("running") -> false
```

## Preconditions

Task status is `running`.

## Steps

1. Set `Status=running`.

## Context

REQUIREMENT leaf: `delete-gate/when-running` (scenario 3).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Status = "running"
	return nil
}
```
