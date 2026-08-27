# Scenario

**Feature**: RunPair wraps missing-fsmonitor Unison output with install hints

```
seed mad-max + Watch + Exec prints "No file monitoring helper program found"
  -> RunPairErr mentions unison-fsmonitor / brew / both sides
  -> state message missing unison-fsmonitor
```

## Preconditions

- Happy doctor; custom Exec exit 3 with Unison fsmonitor error text.
- Watch true (typical --watch path).

## Steps

1. Seed mad-max; Watch true; SkipDoctor optional (happy hooks).
2. Inject Exec that writes helper-missing text and returns 3.
3. Assert rich error + state message.

## Context

- Post-parse wrap so CLI does not stop at bare `unison exit code 3`.

```go
import (
	"context"
	"io"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Mode = "run"
	seedMadMax(req)
	req.Watch = true
	req.Exec = func(ctx context.Context, name string, argv []string, env []string, stdout, stderr io.Writer) (int, error) {
		_ = ctx
		_ = name
		_ = argv
		_ = env
		msg := "Error: No file monitoring helper program found\n"
		if stdout != nil {
			_, _ = io.WriteString(stdout, msg)
		}
		if stderr != nil {
			_, _ = io.WriteString(stderr, msg)
		}
		return 3, nil
	}
	return nil
}
```
