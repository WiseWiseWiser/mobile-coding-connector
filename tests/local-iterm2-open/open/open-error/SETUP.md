# Scenario

**Feature**: Open failure surfaces as 5xx error JSON

```
POST valid dir -> Open returns error -> 5xx {"error":...}
```

## Preconditions

Valid directory; injected Open forced to fail.

## Steps

1. Leaf sets `InjectOpenError`.

## Context

Osascript/open failure path for handlers.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Dir = t.TempDir()
	req.OmitMode = true
	req.OmitSend = true
	return nil
}
```
