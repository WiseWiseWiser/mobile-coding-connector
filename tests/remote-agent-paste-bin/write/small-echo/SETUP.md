# Scenario

**Feature**: small write echoes content on stdout

```
# pipe hi -> paste-bin -> stderr saved 2 bytes + stdout hi
piped "hi" -> PUT scratch -> stderr summary + stdout echo
```

## Preconditions

Scratch reset before write.

## Steps

1. `resetScratch(req)`.
2. `setWritePipe(req, []byte(smallEchoPayload))`.

## Context

REQUIREMENT leaf: `write-small-echo`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	// L3 smoke: paste-bin write echo via product binaries.
	req.UseCLI = true
	resetScratch(req)
	setWritePipe(t, req, []byte(smallEchoPayload))
	return nil
}
```