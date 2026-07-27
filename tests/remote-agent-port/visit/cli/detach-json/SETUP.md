# Scenario

**Feature**: --detach --json prints JSON and exits; session remains

```
port visit PORT --detach --json -> JSON stdout, exit 0, List still has session
```

## Steps

1. Args with --detach --json --provider quick --idle 10m.

```go
import (
	"strconv"
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "cli"
	enableOwnedQuick(req, true, true)
	setCLI(req, "port", "visit", strconv.Itoa(defaultTestPort),
		"--provider", "quick",
		"--idle", "10m",
		"--detach",
		"--json",
	)
	return nil
}
```
