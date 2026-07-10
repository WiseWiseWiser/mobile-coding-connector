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
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Dir = t.TempDir()
	req.OmitMode = true
	req.OmitSend = true
	return nil
}
```
