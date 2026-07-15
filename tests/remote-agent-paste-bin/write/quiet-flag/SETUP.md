# Scenario

**Feature**: quiet flag suppresses stdout echo on small write

```
# pipe hi + -q -> stderr saved, silent stdout
piped hi + --quiet -> no stdout echo
```

## Preconditions

Scratch reset; small payload would normally echo without `-q`.

## Steps

1. `resetScratch(req)`.
2. `setWritePipe(req, []byte(smallEchoPayload), "-q")`.

## Context

REQUIREMENT leaf: `write-quiet-flag`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	resetScratch(req)
	setWritePipe(t, req, []byte(smallEchoPayload), "-q")
	return nil
}
```