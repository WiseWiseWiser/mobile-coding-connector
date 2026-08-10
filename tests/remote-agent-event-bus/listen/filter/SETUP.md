# Scenario

**Feature**: --type filter

```
listen --type T -> only matching Event.Type printed
```

## Steps

1. Leaves set Types and multi-type live seeds.

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
