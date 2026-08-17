# Scenario

**Bug**: `terminal attach session-8` on an `exited` row opens a raw TTY, replays
`[Terminal exited]`, and swallows every keystroke.

```
writer close -> list status=exited
  -> ErrIfSessionNotAttachable
  -> error, DidAttach=false
```

## Steps

1. Set `req.Phase = "exited-refuse"`.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Phase = "exited-refuse"
	return nil
}
```
