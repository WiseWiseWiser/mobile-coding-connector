# Scenario

**Feature**: PublishServer bind address policy

```
# loopback only
StartPublishServer("127.0.0.1:0", ...) -> Addr is loopback
```

## Preconditions

`Op=publish-bind`.

## Steps

1. Set Op publish-bind.
2. Leaf sets ListenAddr.

## Context

REQUIREMENT scenario 5.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	req.Op = "publish-bind"
	return nil
}
```
