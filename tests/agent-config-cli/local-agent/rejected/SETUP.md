# Scenario

**Feature**: local-agent config invalid flags

```
# same rejection rules as remote on local-agent binary
local-agent config <bad> -> non-zero error
```

## Preconditions

None.

## Steps

1. Child sets invalid argv.

## Context

Parity for shared flag validation.

```go
import (
	"testing"
	"time"
)

func Setup(t *testing.T, req *Request) error {
	if req.Timeout <= 0 {
		req.Timeout = 3 * time.Second
	}
	return nil
}
```
