# Scenario

**Feature**: mock script success → service ready

```
GROK_SHOW_USAGE_COMMAND=mock-success.sh -> tty fetch -> status ready + limits
```

## Preconditions

`mock-success.sh` fake TUI emits canonical usage lines after `/usage show`.

## Steps

1. `MockScript=mock-success.sh`.

## Context

REQUIREMENT leaf: `fetch/mock-command-success`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.MockScript = "mock-success.sh"
	return nil
}
```