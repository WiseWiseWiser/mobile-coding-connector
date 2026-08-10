# Scenario

**Feature**: --replay N recent events

```
listen --replay N -> Recent(N) then live stream
```

## Steps

1. Leaves seed RecentEvents + LiveEvents and Replay.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	req.Op = "listen"
	if req.DialMode == "" {
		req.DialMode = "hub"
	}
	return nil
}
```
