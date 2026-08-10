# Scenario

**Feature**: live attach relays binary bytes both ways (FakeAttach echo)

```
seed sess-live + FakeAttach echo
  -> WS attach write "hello-attach\n" -> ReceivedOutput contains same bytes
```

## Preconditions

- Session meta seeded; TTYMode=`live` installs echo FakeAttach via Options.
- No real agent-run child / grok process.

## Steps

1. Seed `sess-live` with terminal_session_id.
2. AttachInput = `hello-attach\n`; AttachHold short.

```go
import (
	"testing"
	"time"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setAttach(req, "sess-live", "live")
	req.Seeds = seedAttachable("sess-live", "term-live")
	req.AttachInput = []byte("hello-attach\n")
	req.AttachHold = 300 * time.Millisecond
	return nil
}
```
