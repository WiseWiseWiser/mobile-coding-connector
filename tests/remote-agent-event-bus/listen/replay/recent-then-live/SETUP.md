# Scenario

**Feature**: replay recent then live

```
# hub pre-seed 2, live 1; --replay 2
listen --replay 2 -> events 1,2 then live 3
```

## Steps

1. Hub; Replay=2; two RecentEvents; one LiveEvent; MaxEvents=3.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	setListenHub(req)
	seedReplayThenLive(req, 2)
	return nil
}
```
