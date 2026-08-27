# Scenario

**Feature**: CLI rejects --interval combined with --watch

```
RunCLI([unison run mad-max --interval 60s --watch])
  -> RunErr; no Exec
```

## Preconditions

- Pair seeded.
- Args include both continuous flags.

## Steps

1. Seed mad-max.
2. Args: `unison run mad-max --interval 60s --watch`.
3. Assert error; Exec not called.

## Context

- FS-event and polled continuous modes are mutually exclusive.

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
	req.FakeExitCode = 0
	req.Args = []string{"unison", "run", "mad-max", "--interval", "60s", "--watch"}
	return nil
}
```
