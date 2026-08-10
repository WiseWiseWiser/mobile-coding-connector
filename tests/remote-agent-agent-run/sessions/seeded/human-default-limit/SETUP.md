# Scenario

**Feature**: default --limit 10 truncates long stores

```
12 seeded metas -> agent-run sessions (no --limit) -> 10 rows + showing note
```

## Preconditions

- 12 sessions on disk (`sess-00` … `sess-11`).
- No `--limit` flag → product default 10.

## Steps

1. Seed 12 sessions via `seedN(12)`.
2. Args = `agent-run sessions`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setCLI(req, "agent-run", "sessions")
	req.Seeds = seedN(12)
	return nil
}
```
