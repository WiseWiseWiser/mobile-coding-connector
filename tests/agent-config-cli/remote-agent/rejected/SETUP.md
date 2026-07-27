# Scenario

**Feature**: remote-agent config invalid flags and combinations

```
# invalid flag / combo -> non-zero exit + error message
remote-agent config <bad> -> stderr/stdout error
```

## Preconditions

No successful dump or UI.

## Steps

1. Child leaves set invalid argv.

## Context

T6–T8.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
	"time"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	// Rejected leaves always use short timeouts; never open browser UI paths long.
	if req.Timeout <= 0 {
		req.Timeout = 3 * time.Second
	}
	return nil
}
```
