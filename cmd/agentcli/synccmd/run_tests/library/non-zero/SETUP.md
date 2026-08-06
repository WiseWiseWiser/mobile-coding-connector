# Scenario

**Feature**: RunPair non-zero Exec exit

```
Exec exit N -> state exitCode N + non-nil error
```

## Preconditions

- Grouping for failed-Exec leaves.

## Steps

1. Ensure Mode is `run`.
2. Leaves set FakeExitCode non-zero.

## Context

- State still written after failed Unison exit.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	if req.Mode == "" {
		req.Mode = "run"
	}
	return nil
}
```
