# Scenario

**Feature**: abnormal remote close yields clear attach error

```
sess-live + FakeAttach unclean close after ~80ms
  -> AttachErr set / ExitCode != 0 (restore path when product wires OnLocalRestore)
```

## Preconditions

- Not a normal WS close (1000/1001).
- Product client should restore local TTY; L2 records `TermRestored` when
  Options.OnLocalRestore is invoked (optional soft assert).

## Steps

1. Seed sess-live; TTYMode=`unclean`; short AttachHold.

```go
import (
	"testing"
	"time"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setAttach(req, "sess-live", "unclean")
	req.Seeds = seedAttachable("sess-live", "term-live")
	req.AttachHold = 80 * time.Millisecond
	return nil
}
```
