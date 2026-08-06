# Scenario

**Feature**: Status reports pair + serve + last-run state

```
operator -> synccmd.Status(opts{Name, ServeOK}) -> StatusReport
```

## Preconditions

- Grouping for named and all-pairs status.
- State files optional under `{StoreDir}/state/`.

## Steps

1. Default Mode to `status`.
2. Default ServeOK to up when unset by leaf.
3. Leaves seed pairs and optional state.

## Context

- Library asserts; no real run history until P3 writes state.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	if req.Mode == "" {
		req.Mode = "status"
	}
	if req.ServeOK == nil {
		req.ServeOK = serveUp()
	}
	return nil
}
```
