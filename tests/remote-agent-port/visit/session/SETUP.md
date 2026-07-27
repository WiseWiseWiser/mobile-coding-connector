# Scenario

**Feature**: visit session uniqueness, list, stop, not-listening warn

```
Start / List / Stop / CLI visit not-listening -> error | sessions | warning:
```

## Preconditions

Active sessions are in-memory only.

## Steps

1. Leaf chooses Op (duplicate/list/stop/cli).
2. Run manager or CLI.
3. Assert errors, list contents, or stderr warning.

## Context

Same port twice while active → error; stop by id or port.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Port = defaultTestPort
	enableOwnedQuick(req, true, true)
	return nil
}
```
