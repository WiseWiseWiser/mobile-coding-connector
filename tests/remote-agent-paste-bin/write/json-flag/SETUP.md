# Scenario

**Feature**: write with --json outputs PUT response only

```
# pipe payload + --json -> stdout JSON only (no preview/echo)
piped hi + --json -> PUT response JSON on stdout
```

## Preconditions

Scratch reset before write.

## Steps

1. `resetScratch(req)`.
2. `setWritePipe(req, []byte(smallEchoPayload), "--json")`.

## Context

REQUIREMENT leaf: `write-json-flag`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	resetScratch(req)
	setWritePipe(t, req, []byte(smallEchoPayload), "--json")
	return nil
}
```