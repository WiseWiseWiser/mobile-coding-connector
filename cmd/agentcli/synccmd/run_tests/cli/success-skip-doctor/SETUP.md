# Scenario

**Feature**: CLI run --skip-doctor succeeds with fake Exec despite serve down

```
seed mad-max + serve down
  -> RunCLI([unison run mad-max --skip-doctor])
  -> Exec called; RunErr empty; state exit 0
```

## Preconditions

- Pair + profile; ServeOK fails; FakeExitCode 0.
- Args include `--skip-doctor`.

## Steps

1. Seed mad-max; ServeOK down; SkipDoctor implied by Args.
2. Args: `unison run mad-max --skip-doctor`.
3. Assert CLI success + state.

## Context

- End-to-end CLI wire for P3 without real Unison.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Mode = "cli"
	seedMadMax(req)
	req.ServeOK = serveDown("connection refused")
	req.FakeExitCode = 0
	req.Args = []string{"unison", "run", "mad-max", "--skip-doctor"}
	return nil
}
```
