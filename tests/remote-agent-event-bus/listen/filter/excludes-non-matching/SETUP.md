# Scenario

**Feature**: type filter excludes non-matching

```
# two live events; filter seatalk only
listen --type seatalk.message.received -> only seatalk line
```

## Steps

1. Hub; two live events different types; Types=[seatalk]; MaxEvents=1 (one printed).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	setListenHub(req)
	seedTwoLiveDifferentTypes(req)
	req.Types = []string{fixtureTypeSeatalk}
	// Stop after one *printed* matching event; non-matching must not count.
	req.MaxEvents = 1
	return nil
}
```
