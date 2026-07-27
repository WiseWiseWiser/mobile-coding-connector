# Scenario

**Feature**: bare local-agent config prints help (does not start UI)

```
# bare config -> help with local-agent name; no Config UI
local-agent config -> stdout help
```

## Preconditions

Empty HOME.

## Steps

1. Args = `["config"]`.

## Context

local-agent parity T1.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Args = []string{"config"}
	return nil
}
```
