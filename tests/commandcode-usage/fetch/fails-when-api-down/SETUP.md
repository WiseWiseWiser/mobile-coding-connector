# Scenario

**Feature**: an unreachable provider API reports the network error

```
closed fixture API -> status error: Network error: unable to reach API
```

## Preconditions

1. The fixture server is closed before the fetch.

## Steps

1. Set `Op=api-down`.

## Context

Offline machines must show a network error, not a credential error.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "api-down"
	return nil
}
```
