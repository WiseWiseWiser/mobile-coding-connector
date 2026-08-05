# Scenario

**Feature**: FileSessionStore — on-disk session JSON under injectable Root

```
# store ops under {Root}/ssh-sessions/{profileID}.json
tests -> FileSessionStore.Save / Load / Clear
  -> Session fields round-trip; missing nil; corrupt error; dead PID !Alive
```

## Preconditions

- Scenario family is session-store (leaf sets concrete Scenario).
- Root is absolute under d.DOCTEST_CASE (set by root Setup).

## Steps

1. Grouping does not override Scenario; leaves choose save-load | missing | clear | dead-pid | corrupt.

## Context

- Implements `SessionStore.Load` plus Save/Clear mutators for serve.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	_ = req
	// Leaves set Scenario + SessionToSave as needed.
	return nil
}
```
