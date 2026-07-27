# Scenario

**Feature**: local-agent --show --web mutual exclusion

```
local-agent config --show --web -> non-zero error
```

## Preconditions

None.

## Steps

1. Args = `config --show --web`.

## Context

Parity T7.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Args = []string{"config", "--show", "--web"}
	return nil
}
```
