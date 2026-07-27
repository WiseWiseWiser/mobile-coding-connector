# Scenario

**Feature**: Run Now disabled while task is running

```
CanRunCronTask("running") -> false
```

## Preconditions

Task status is `running`.

## Steps

1. Set `Status=running`.

## Context

REQUIREMENT leaf: `action/run-when-running`.

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
