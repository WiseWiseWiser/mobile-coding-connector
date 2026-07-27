# Scenario

**Feature**: local-agent --json without --show

```
local-agent config --json -> non-zero, mentions --show
```

## Preconditions

None.

## Steps

1. Args = `config --json`.

## Context

Parity T6.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Args = []string{"config", "--json"}
	return nil
}
```
