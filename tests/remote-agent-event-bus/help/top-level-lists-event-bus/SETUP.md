# Scenario

**Feature**: top-level help lists event-bus

```
remote-agent --help -> Usage includes event-bus command
```

## Steps

1. Args: `--help` (top-level).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	setCLI(req, "--help")
	return nil
}
```
