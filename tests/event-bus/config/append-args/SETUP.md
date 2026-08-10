# Scenario

**Feature**: AppendEventBusServerArgs for keep-alive child argv

```
# forward non-default port/token and no-publish
AppendEventBusServerArgs(base, cfg) -> argv with flags
```

## Preconditions

`Op=append-args`.

## Steps

1. Set Op append-args.
2. Leaf sets BaseArgs + flag fields.

## Context

REQUIREMENT scenario 8.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	req.Op = "append-args"
	if req.BaseArgs == nil {
		req.BaseArgs = []string{"ai-critic-server", "--port", "8080"}
	}
	return nil
}
```
