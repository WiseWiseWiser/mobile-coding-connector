# Scenario

**Feature**: attach when TTY is unreachable fails clearly

```
meta exists + ResolveTTY(reachable=false) -> attach -> clear error (no hang)
```

## Preconditions

- Session meta seeded with `terminal_session_id`.
- Injected ResolveTTY reports unreachable (no live listen addr).

## Steps

1. Seed attachable session `sess-dead`.
2. TTYMode = `unreachable`; Op=attach.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setAttach(req, "sess-dead", "unreachable")
	req.Seeds = seedAttachable("sess-dead", "term-dead")
	return nil
}
```
