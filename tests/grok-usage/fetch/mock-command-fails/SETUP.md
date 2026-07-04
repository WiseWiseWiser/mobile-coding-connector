# Scenario

**Feature**: mock script failure → service error status

```
GROK_SHOW_USAGE_COMMAND=mock-fail.sh -> tty fetch error -> status error
```

## Preconditions

`mock-fail.sh` fake TUI writes stderr and exits 1 after prompt.

## Steps

1. `MockScript=mock-fail.sh`.

## Context

REQUIREMENT leaf: `fetch/mock-command-fails`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.MockScript = "mock-fail.sh"
	return nil
}
```