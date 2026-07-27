# Scenario

**Feature**: nested Status menu with Enable and Disable only as children

```
Backup -> Status: … ▸ Enable | Disable
```

## Preconditions

Status title carries state; Enable/Disable are the only nested actions under Status.

## Steps

1. Set `ClientLeaf=status-enable-disable`.

## Context

REQUIREMENT #25.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.ClientLeaf = "status-enable-disable"
	return nil
}
```
