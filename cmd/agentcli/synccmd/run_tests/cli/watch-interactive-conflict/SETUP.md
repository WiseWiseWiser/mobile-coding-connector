# Scenario

**Feature**: CLI rejects --watch combined with --interactive

```
RunCLI([unison run mad-max --watch --interactive])
  -> RunErr mentions incompatible flags; no Exec
```

## Preconditions

- Pair seeded (not required for flag gate, but keeps resolve happy if reached).
- Args include both `--watch` and `--interactive`.

## Steps

1. Seed mad-max.
2. Args: `unison run mad-max --watch --interactive`.
3. Assert error; Exec not called.

## Context

- Unattended watch requires batch; interactive prompts conflict.

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
	req.Args = []string{"unison", "run", "mad-max", "--watch", "--interactive"}
	return nil
}
```
