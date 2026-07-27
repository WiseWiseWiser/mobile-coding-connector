# Scenario

**Feature**: remote-agent config help and bare invocation

```
# bare / --help / -h print help; no Config UI
remote-agent config [help flags] -> stdout help, exit 0
```

## Preconditions

No config file required.

## Steps

1. Child leaves choose bare vs help flags.

## Context

Target: bare no longer opens the web UI.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	// help group: no seed
	req.SeedConfig = nil
	return nil
}
```
