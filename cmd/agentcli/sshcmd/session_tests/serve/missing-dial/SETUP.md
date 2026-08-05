# Scenario

**Feature**: Start without Dial configured returns a clear error

```
# misconfiguration
ServeService{Dial: nil}.Start(ctx) -> error (dial not configured)
```

## Preconditions

- Scenario: `serve-missing-dial`.
- Dial is intentionally nil; Store and ProfileID still set.

## Steps

1. Construct ServeService with nil Dial.
2. Call Start with a short timeout context.
3. Assert ServeErr is non-empty and mentions dial / not configured (case-insensitive).

## Context

- Fail-fast before listen/save when Dial is required for the tunnel remote end.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Scenario = ScenarioServeMissingDial
	return nil
}
```
