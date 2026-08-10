# Scenario

**Feature**: JSON list when store has zero sessions

```
empty store -> agent-run sessions --json -> {"sessions":[]}
```

## Preconditions

- No session seeds.
- Machine JSON only (no ANSI table).

## Steps

1. Args = `agent-run sessions --json`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setCLI(req, "agent-run", "sessions", "--json")
	req.Seeds = nil
	return nil
}
```
