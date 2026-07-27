# Scenario

**Feature**: visit when port not listening warns but continues

```
port visit <port> --detach --json + NotListening -> warning: on stderr, exit 0, session started
```

## Steps

1. CLI visit with NotListening; both providers available.

```go
import (
	"strconv"
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "cli"
	req.NotListening = true
	enableOwnedQuick(req, true, true)
	// short idle + detach so CLI returns
	setCLI(req, "port", "visit", strconv.Itoa(defaultTestPort),
		"--provider", "quick",
		"--idle", "30s",
		"--detach",
		"--json",
	)
	return nil
}
```
