# Scenario

**Feature**: mock script failure → service error status

```
mock-fail.sh exit 1 -> status error
```

## Preconditions

`mock-fail.sh` writes stderr and exits 1.

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