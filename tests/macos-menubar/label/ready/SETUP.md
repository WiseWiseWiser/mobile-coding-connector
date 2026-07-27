# Scenario

**Feature**: ready status shows weekly limit

```
FormatGrokLabel("ready","6%","") -> "Grok 6%"
```

## Preconditions

Status ready with weekly limit present.

## Steps

1. Set inputs per requirement table.

## Context

REQUIREMENT leaf: `label/ready`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Status = "ready"
	req.WeeklyLimit = "6%"
	req.ErrorMsg = ""
	return nil
}
```