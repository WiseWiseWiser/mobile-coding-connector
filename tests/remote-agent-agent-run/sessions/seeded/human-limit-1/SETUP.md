# Scenario

**Feature**: --limit 1 caps list to one session

```
3 seeded metas -> agent-run sessions --limit 1 -> one row + showing note
```

## Preconditions

- Three sessions on disk.
- Explicit `--limit 1`.

## Steps

1. Seed three ordered sessions.
2. Args = `agent-run sessions --limit 1`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setCLI(req, "agent-run", "sessions", "--limit", "1")
	req.Seeds = seedThreeOrdered()
	return nil
}
```
