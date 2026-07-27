# Scenario

**Feature**: read empty scratch pad

```
# no scratch.json -> GET returns empty content -> silent stdout
delete scratch.json -> remote-agent paste-bin -> (no stdout)
```

## Preconditions

`scratch.json` absent under `configHome/file-transfer/`.

## Steps

1. `deleteScratch(req)`.
2. `setReadTTY(req)` — default read, TTY stdin.

## Context

REQUIREMENT leaf: `read-empty`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	deleteScratch(req)
	setReadTTY(t, req)
	return nil
}
```