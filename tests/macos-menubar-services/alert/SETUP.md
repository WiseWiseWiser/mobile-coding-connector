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
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "alert"
	return nil
}
```