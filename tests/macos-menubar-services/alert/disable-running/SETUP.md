# Scenario

**Feature**: disable alert while service is running

```
DisableAlertMessage(true) -> msgDisableRunning
```

## Preconditions

User disables a service that still has a live process.

## Steps

1. Set `Running=true`.

## Context

REQUIREMENT leaf: `alert/disable-running`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Running = true
	return nil
}
```