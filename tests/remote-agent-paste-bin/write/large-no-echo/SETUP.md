# Scenario

**Feature**: large write suppresses stdout echo

```
# pipe 5000 x bytes -> stderr preview truncated, silent stdout
piped 5000 bytes -> PUT scratch -> preview only on stderr
```

## Preconditions

Scratch reset; payload is 5000 ASCII `x` bytes (> 4096 echo threshold).

## Steps

1. `resetScratch(req)`.
2. `setWritePipe(req, repeatByte('x', largePayloadSize))`.

## Context

REQUIREMENT leaf: `write-large-no-echo`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	resetScratch(req)
	setWritePipe(t, req, repeatByte('x', largePayloadSize))
	return nil
}
```