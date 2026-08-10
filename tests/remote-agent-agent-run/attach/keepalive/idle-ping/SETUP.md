# Scenario

**Feature**: idle attach receives Ping within injected interval

```
sess-live + PingInterval=50ms + FakeAttach hold
  -> dial attach, idle wait ~400ms -> PingCount >= 1
```

## Preconditions

- Zero application data traffic during wait.
- Server (or negotiated hop) emits Ping control frames.

## Steps

1. Seed attachable session.
2. `setKeepalive(sess-live, 50ms, 400ms)`.

```go
import (
	"testing"
	"time"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setKeepalive(req, "sess-live", 50*time.Millisecond, 400*time.Millisecond)
	req.Seeds = seedAttachable("sess-live", "term-live")
	return nil
}
```
