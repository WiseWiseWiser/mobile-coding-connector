# Scenario

**Feature**: concurrent refresh does not double-fetch

```
two concurrent refresh -> GROK_MOCK_COUNTER_FILE == 1
```

## Preconditions

`mock-slow.sh` fake TUI with `GROK_MOCK_COUNTER_FILE` side-effect.

## Steps

1. `MockScript=mock-slow.sh`.

## Context

REQUIREMENT leaf: `refresh/skips-overlap`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.MockScript = "mock-slow.sh"
	return nil
}
```