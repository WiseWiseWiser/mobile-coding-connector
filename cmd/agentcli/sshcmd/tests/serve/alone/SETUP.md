# Scenario

**Feature**: `--serve` alone starts the serve session

```
# happy serve
operator -> sshcmd.Run(Args=["--serve"], Serve=mock)
  -> ModeServe; ServeStarter.Start once with ProfileID; Runner not called
```

## Preconditions

- Args: `["--serve"]` only.
- Mock `ServeStarter` returns nil.

## Steps

1. Set Args to `["--serve"]`.
2. Run; Assert Start called once and Runner never called.

## Context

- P1 serve is a mockable hook only (no real tunnel).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Args = []string{"--serve"}
	return nil
}
```
