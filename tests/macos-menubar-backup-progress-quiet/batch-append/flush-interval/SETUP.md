# Scenario

**Feature**: batch flush interval is in the 100–200 ms band

```
ProgressSession
  -> Timer / asyncAfter / flushInterval ~ 0.15s | 150ms
  # band: 100–200ms inclusive
```

## Preconditions

Source shows a flush timer or interval constant in band (e.g. `0.15`, `150`,
`.milliseconds(150)` near Timer/flush).

## Steps

1. ClientLeaf=flush-interval.

## Context

REQUIREMENT #4; RED until interval/timer exists (current immediate append has none).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.ClientLeaf = "flush-interval"
	return nil
}
```
