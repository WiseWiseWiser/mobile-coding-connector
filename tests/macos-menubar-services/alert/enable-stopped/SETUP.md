# Scenario

**Feature**: enable alert while service is stopped

```
EnableAlertMessage(false) -> msgEnableStopped
```

## Preconditions

User enables a stopped service; daemon start is deferred.

## Steps

1. Set `Running=false`.

## Context

REQUIREMENT leaf: `alert/enable-stopped`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Running = false
	return nil
}
```