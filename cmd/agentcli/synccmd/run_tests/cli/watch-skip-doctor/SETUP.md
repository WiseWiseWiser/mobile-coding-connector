# Scenario

**Feature**: CLI run --watch --skip-doctor passes -repeat watch to Exec

```
seed mad-max + serve down
  -> RunCLI([unison run mad-max --watch --skip-doctor])
  -> Exec argv has -repeat watch; RunErr empty
```

## Preconditions

- Pair + profile; ServeOK fails; FakeExitCode 0.
- Args include `--watch` and `--skip-doctor`.

## Steps

1. Seed mad-max; ServeOK down.
2. Args: `unison run mad-max --watch --skip-doctor`.
3. Assert CLI success + Exec argv.

## Context

- End-to-end CLI wire for `--watch` without real Unison.

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
	req.Args = []string{"unison", "run", "mad-max", "--watch", "--skip-doctor"}
	return nil
}
```
