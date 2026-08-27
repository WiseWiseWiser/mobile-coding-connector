# Scenario

**Feature**: CLI rejects bare --interval without time unit

```
RunCLI([unison run mad-max --interval 60 --skip-doctor])
  -> RunErr mentions invalid / unit; no Exec
```

## Preconditions

- Pair seeded.
- Args use bare `60` (Go ParseDuration requires a unit).

## Steps

1. Seed mad-max.
2. Args: `unison run mad-max --interval 60 --skip-doctor`.
3. Assert parse error; Exec not called.

## Context

- Same duration style as cron `--every` (`60s`, `5m`, `1h`).

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
	req.Args = []string{"unison", "run", "mad-max", "--interval", "60", "--skip-doctor"}
	return nil
}
```
