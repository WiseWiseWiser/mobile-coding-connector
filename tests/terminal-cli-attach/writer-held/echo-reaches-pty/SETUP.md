# Scenario

**Bug**: `terminal attach` shows the prompt but typed input never reaches the PTY
when the web UI (or any writer) is still connected.

```
# crime scene (session-9)
writer WS held
  -> CLI attach (TerminalAttachConnectOptions)
  -> echo CLI_ATTACH_MARKER
  -> MARKER must appear in attach stdout
```

## Steps

1. Set `req.Phase = "writer-held-echo"` and a unique marker.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Phase = "writer-held-echo"
	req.Marker = "CLI_ATTACH_MARKER"
	return nil
}
```
