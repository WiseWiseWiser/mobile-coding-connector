# Scenario

**Feature**: RunPair --skip-doctor runs Exec even when serve is down

```
seed mad-max + ServeOK fail + SkipDoctor true
  -> Exec called; exit 0; state written
```

## Preconditions

- ServeOK fails; SkipDoctor true; FakeExitCode 0.

## Steps

1. Seed mad-max; override ServeOK down; SkipDoctor true.
2. Assert Exec called and state exit 0.

## Context

- Operator override path; doctor not required.

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
	req.SkipDoctor = true
	req.ServeOK = serveDown("connection refused")
	req.FakeExitCode = 0
	return nil
}
```
