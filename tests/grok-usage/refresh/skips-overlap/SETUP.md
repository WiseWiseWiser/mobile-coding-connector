# Scenario

**Feature**: concurrent refresh does not double-fetch

```
two concurrent refresh -> FetchInvocationCount == 1
```

## Preconditions

Injectable slow fetcher increments a counter once while holding `fetching`.

## Steps

1. `FetchMode=slow` (via refresh Run harness).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.FetchMode = "slow"
	return nil
}
```
