# Scenario

**Feature**: mock script success → service ready

```
mock-success.sh -> status ready + limits
```

## Preconditions

`mock-success.sh` emits canonical lines.

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