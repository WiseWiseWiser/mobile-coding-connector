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
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Args = []string{"config"}
	return nil
}
```
