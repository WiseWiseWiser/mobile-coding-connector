# Scenario

**Feature**: RunPair library path with injectable Exec

```
operator -> RunPair(opts{Name, Exec, SkipDoctor, probes})
  -> optional doctor gate -> Exec -> state/<name>.json
```

## Preconditions

- Grouping for RunPair leaves.
- Leaves seed pairs, hooks, FakeExitCode / SkipDoctor.

## Steps

1. Default Mode to `run` when empty.
2. Leaves configure pair, Exec outcome, doctor gate.

## Context

- Library-first asserts; CLI covered under `cli/`.

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
