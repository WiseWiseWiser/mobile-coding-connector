# Scenario

**Feature**: sessions listed newest-first by updated_at

```
sess-old/mid/new with staggered updated_at
  -> agent-run sessions --json --limit 0
  -> [sess-new, sess-mid, sess-old]
```

## Preconditions

- Three seeds with distinct `updated_at` (2020 / 2021 / 2022).
- Sort: `updated_at` desc, then `created_at` desc, then `session_id` asc
  (local agent-run parity).

## Steps

1. Seed three ordered sessions.
2. Args = `agent-run sessions --json --limit 0`.

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
