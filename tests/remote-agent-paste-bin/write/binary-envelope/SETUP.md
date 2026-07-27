# Scenario

**Feature**: binary stdin stored as b64 envelope

```
# pipe bytes with NUL -> PUT paste-bin:b64:... -> round-trip raw bytes
piped binary -> envelope on server -> decode matches stdin
```

## Preconditions

Scratch reset; payload contains embedded NUL byte.

## Steps

1. `resetScratch(req)`.
2. `setWritePipe(req, []byte(binaryEnvelopePayload))`.

## Context

REQUIREMENT leaf: `write-binary-envelope`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	resetScratch(req)
	setWritePipe(t, req, []byte(binaryEnvelopePayload))
	return nil
}
```