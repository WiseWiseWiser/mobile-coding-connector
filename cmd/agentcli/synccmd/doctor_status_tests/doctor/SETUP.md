# Scenario

**Feature**: Doctor readiness checks for one Unison pair

```
operator -> synccmd.Doctor(opts{Name, probes}) -> DoctorReport + error?
```

## Preconditions

- Grouping for doctor library and CLI leaves.
- Leaves seed pairs.json and probe hooks.

## Steps

1. Default Mode to `doctor` when empty.
2. Leaves set PairName, seeds, fakes.

## Context

- Library-first asserts; one CLI fail-exit leaf under `cli/`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	if req.Mode == "" {
		req.Mode = "doctor"
	}
	return nil
}
```
