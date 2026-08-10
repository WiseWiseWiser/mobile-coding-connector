# Scenario

**Feature**: loopback HTTP PublishServer

```
# POST /publish on 127.0.0.1 with optional Bearer token
StartPublishServer(addr, hub, opts) -> POST /publish
```

## Preconditions

Publish server binds loopback; hub is in-process.

## Steps

1. Default Op publish-http; bind/conflict groups override.
2. Leaves set ServerToken / ClientToken / Event.

## Context

REQUIREMENT scenarios 3–5, 9.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	if req.Op == "" {
		req.Op = "publish-http"
	}
	if req.ListenAddr == "" {
		req.ListenAddr = "127.0.0.1:0"
	}
	return nil
}
```
