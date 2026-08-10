# Scenario

**Feature**: event-bus root help lists listen

```
remote-agent event-bus --help -> Usage: … listen …
```

## Steps

1. Args: `event-bus --help`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	setCLI(req, "event-bus", "--help")
	return nil
}
```
