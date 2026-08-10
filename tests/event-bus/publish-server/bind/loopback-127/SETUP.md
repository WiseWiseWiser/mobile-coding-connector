# Scenario

**Feature**: server listens on 127.0.0.1

```
# bind loopback ephemeral port
ListenAddr=127.0.0.1:0 -> Addr host is loopback
```

## Steps

1. ListenAddr `127.0.0.1:0`.
2. Assert IsLoopback.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	req.ListenAddr = "127.0.0.1:0"
	req.ServerToken = ""
	return nil
}
```
