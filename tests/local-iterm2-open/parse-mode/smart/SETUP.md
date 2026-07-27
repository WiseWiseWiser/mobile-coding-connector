# Scenario

**Feature**: mode "smart" maps to ModeSmart

```
ParseOpenMode("smart") -> ModeSmart
```

## Preconditions

None beyond parse group.

## Steps

1. Set `ModeInput=smart`.

## Context

JSON `"smart"` → `ModeSmart`.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.ModeInput = "smart"
	return nil
}
```
