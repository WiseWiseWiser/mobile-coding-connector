# Scenario

**Feature**: RunPair doctor gate before Exec

```
doctor fail + !SkipDoctor -> abort (no Exec)
SkipDoctor + serve down -> Exec still runs
```

## Preconditions

- Grouping for skip-doctor vs doctor-fail leaves.

## Steps

1. Ensure Mode is `run`.
2. Leaves set SkipDoctor and ServeOK.

## Context

- Gate is pre-Exec only; state not written on abort.

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
