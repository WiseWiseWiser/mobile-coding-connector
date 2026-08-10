# Scenario

**Feature**: unknown event-bus subcommand

```
event-bus foo -> Error: unknown … + non-zero
```

## Steps

1. Args: `event-bus foo`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	setCLI(req, "event-bus", "foo")
	return nil
}
```
