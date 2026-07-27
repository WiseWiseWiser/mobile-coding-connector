# Scenario

**Feature**: empty Terminals list placeholder

```
FormatTerminalsEmptyLabel() -> "No terminal sessions"
```

## Preconditions

`Op=empty` dispatches to `menubar.FormatTerminalsEmptyLabel`.

## Steps

1. Leaf invokes empty-label formatter (no extra inputs).

## Context

REQUIREMENT: empty terminals label string sealed in leaf.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "empty"
	return nil
}
```
