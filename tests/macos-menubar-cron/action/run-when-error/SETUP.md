# Scenario

**Feature**: Run Now enabled when task is in error

```
CanRunCronTask("error") -> true
```

## Preconditions

Task status is `error`.

## Steps

1. Set `Status=error`.

## Context

REQUIREMENT leaf: `action/run-when-error`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Status = "error"
	return nil
}
```
