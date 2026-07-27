# Scenario

**Feature**: unreachable status mentions host and retry/test

```
FormatStatus(unreachable, "https://example.com") -> cannot reach host + retry/test guidance
```

## Preconditions

Network/host unreachable for the resolved server.

## Steps

1. Set `ConnectionState=unreachable` and server URL.

## Context

REQUIREMENT leaf: `status/unreachable`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.ConnectionState = "unreachable"
	req.StatusServer = "https://example.com"
	req.Token = "must-not-appear"
	return nil
}
```
