# Scenario

**Feature**: CLI --interval 60s passes -repeat 60 to Exec

```
seed mad-max + serve down
  -> RunCLI([unison run mad-max --interval 60s --skip-doctor])
  -> Exec argv has -repeat 60; RunErr empty
```

## Preconditions

- Pair + profile; ServeOK fails; FakeExitCode 0.
- Args include `--interval 60s` and `--skip-doctor`.

## Steps

1. Seed mad-max; ServeOK down.
2. Args: `unison run mad-max --interval 60s --skip-doctor`.
3. Assert CLI success + Exec argv.

## Context

- Duration parse → Unison seconds for continuous poll mode.

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
	req.Args = []string{"unison", "run", "mad-max", "--interval", "60s", "--skip-doctor"}
	return nil
}
```
