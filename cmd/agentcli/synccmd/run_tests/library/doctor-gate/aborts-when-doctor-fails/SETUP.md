# Scenario

**Feature**: RunPair aborts before Exec when doctor fails and skip-doctor false

```
seed mad-max + ServeOK fail + SkipDoctor false
  -> RunPair error; Exec not called; no state file
```

## Preconditions

- Pair + profile seeded.
- ServeOK fails; other hooks happy.
- SkipDoctor false.

## Steps

1. Seed mad-max with happy hooks then override ServeOK to fail.
2. SkipDoctor false.
3. Assert no Exec and no state.

## Context

- Critical safety gate for operator run without --skip-doctor.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Mode = "run"
	seedMadMax(req)
	req.SkipDoctor = false
	req.ServeOK = serveDown("connection refused")
	req.FakeExitCode = 0
	return nil
}
```
