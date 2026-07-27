# Scenario

**Feature**: bare remote-agent config prints help (does not start UI)

```
# bare config -> help on stdout, exit 0; never "Config UI running"
remote-agent config -> stdout help
```

## Preconditions

Empty HOME config directory (no seed).

## Steps

1. Args = `["config"]` only.

## Context

T1: bare config must not block on browser UI.

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
