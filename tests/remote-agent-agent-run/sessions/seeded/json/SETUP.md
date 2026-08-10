# Scenario

**Feature**: --json machine list of seeded sessions

```
3 seeded metas -> agent-run sessions --json -> {"sessions":[{session_id,runner,status,...}]}
```

## Preconditions

- Three ordered sessions on disk.
- JSON mode only.

## Steps

1. Seed three ordered sessions.
2. Args = `agent-run sessions --json --limit 0` (all).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setCLI(req, "agent-run", "sessions", "--json", "--limit", "0")
	req.Seeds = seedThreeOrdered()
	return nil
}
```
