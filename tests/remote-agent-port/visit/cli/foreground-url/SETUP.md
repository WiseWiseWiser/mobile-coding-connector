# Scenario

**Feature**: foreground visit prints URL and provider

```
port visit --idle 50ms --detach=false  OR short idle with detach false
```

Foreground normally blocks; harness uses `--detach` false with very short idle
when product supports idle exit, OR tests use `--json --detach` alternative.

For L2 without blocking forever: use `--idle 1s --detach` is wrong for this leaf.
This leaf uses `--detach` **false** conceptually but Run must not hang.

Locked approach: CLI with `--idle 100ms` without detach; product exits when idle
fires. Harness sets provider quick and waits via agentcli return.

## Steps

1. Args: port visit PORT --provider quick --idle 100ms (foreground).

```go
import (
	"strconv"
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "cli"
	enableOwnedQuick(req, true, true)
	// Foreground: no --detach. Short idle so Run returns when session expires.
	setCLI(req, "port", "visit", strconv.Itoa(defaultTestPort),
		"--provider", "quick",
		"--idle", "100ms",
	)
	return nil
}
```
