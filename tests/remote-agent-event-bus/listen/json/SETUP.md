# Scenario

**Feature**: --json NDJSON event stream

```
listen --json -> one Event JSON object per stdout line (no ANSI)
```

## Steps

1. JSON=true.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	req.JSON = true
	return nil
}
```
