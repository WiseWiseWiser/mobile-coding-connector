# Scenario

**Feature**: --limit 0 lists all sessions

```
12 seeded metas -> agent-run sessions --limit 0 -> all 12 rows, no truncation note
```

## Preconditions

- 12 sessions on disk.
- `--limit 0` means all (local agent-run parity).

## Steps

1. Seed 12 sessions.
2. Args = `agent-run sessions --limit 0`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setCLI(req, "agent-run", "sessions", "--limit", "0")
	req.Seeds = seedN(12)
	return nil
}
```
