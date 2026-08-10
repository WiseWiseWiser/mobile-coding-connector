# Scenario

**Feature**: human list when store has zero sessions

```
empty store -> agent-run sessions -> header-only or empty-friendly message
```

## Preconditions

- No session seeds.
- Human output (no `--json`).

## Steps

1. Args = `agent-run sessions`.
2. Seeds remain empty.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setCLI(req, "agent-run", "sessions")
	req.Seeds = nil
	return nil
}
```
