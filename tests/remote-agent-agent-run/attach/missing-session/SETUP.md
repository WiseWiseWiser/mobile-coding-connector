# Scenario

**Feature**: attach to unknown session fails clearly

```
empty store -> WS attach sess-nope -> 404 / error (no hang)
```

## Preconditions

- No seeds (session absent from agentstorage).
- Op=attach dials the attach endpoint directly (L2).

## Steps

1. Seeds empty; AttachSessionID = `sess-nope`; TTYMode empty/missing.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setAttach(req, "sess-nope", "missing")
	req.Seeds = nil
	return nil
}
```
