# Scenario

**Feature**: disable/enable NSAlert message copy

```
running state -> DisableAlertMessage / EnableAlertMessage -> server constants
```

## Preconditions

`Op=alert` mirrors `server/services` `msgDisableRunning` and `msgEnableStopped`.

## Steps

1. Leaf sets `Running` for the alert scenario.

## Context

REQUIREMENT section A — alert message leaves.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "alert"
	return nil
}
```