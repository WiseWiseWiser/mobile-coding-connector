# Scenario

**Feature**: human pretty event lines

```
listen (no --json) -> green connected + HH:MM:SS type lines
```

## Steps

1. JSON=false; hub mode default.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	req.JSON = false
	return nil
}
```
