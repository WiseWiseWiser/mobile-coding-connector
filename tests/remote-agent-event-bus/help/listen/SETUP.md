# Scenario

**Feature**: event-bus listen help documents flags

```
remote-agent event-bus listen --help -> flags --type --json --replay
```

## Steps

1. Args: `event-bus listen --help`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	setCLI(req, "event-bus", "listen", "--help")
	return nil
}
```
