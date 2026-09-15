# Scenario

**Feature**: a ready fetch reports the plan, cycle usage, windows, and usage URL

```
fixture API -> status ready, GOAT/active, 5%, $66.19, 1,870 requests, 26 days, 15%/11%
```

## Preconditions

1. The fixture API reports an active `individual-goat` subscription with 5% of the cycle used.

## Steps

1. Set `Op=ready`.

## Context

This is the account shape the user sees: GOAT plan, a cycle credit allowance, two windows.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "ready"
	return nil
}
```
