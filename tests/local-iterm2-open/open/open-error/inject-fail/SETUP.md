# Scenario

**Feature**: injected Open error becomes HTTP 5xx

```
Open returns "osascript boom" -> 5xx {"error":...}
```

## Preconditions

Valid dir from parent.

## Steps

1. Set `InjectOpenError=osascript boom`.

## Context

REQUIREMENT scenario 5 inverse (failure).

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.InjectOpenError = "osascript boom"
	return nil
}
```
