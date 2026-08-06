# Scenario

**Feature**: RunPair success writes state with exitCode 0

```
seed mad-max + happy doctor + FakeExitCode 0
  -> RunPair -> Exec called; state lastRunAt + exitCode 0; nil error
```

## Preconditions

- Pair + profile; doctor hooks all ok.
- SkipDoctor false (doctor must pass).
- FakeExitCode 0.

## Steps

1. Seed mad-max and happy hooks.
2. FakeExitCode 0.
3. Assert Exec called, state file, RunPairErr empty.

## Context

- Primary happy path for implementer.

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
	req.FakeExitCode = 0
	return nil
}
```
