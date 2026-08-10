# Scenario

**Feature**: second StartPublishServer on occupied port fails

```
# port contention
first Start ok; second Start same host:port -> non-nil error
```

## Steps

1. Run publish-port-in-use (harness picks free port, starts twice).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	req.ServerToken = ""
	return nil
}
```
