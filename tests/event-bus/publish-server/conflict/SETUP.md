# Scenario

**Feature**: hard-fail when publish port already in use

```
# second StartPublishServer on same addr fails
Start(addr) ok; Start(addr) again -> error
```

## Preconditions

`Op=publish-port-in-use`.

## Steps

1. Set Op publish-port-in-use.

## Context

REQUIREMENT scenario 9.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	req.Op = "publish-port-in-use"
	return nil
}
```
