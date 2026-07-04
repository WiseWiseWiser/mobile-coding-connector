# Scenario

**Feature**: long error message truncated for menu bar

```
FormatGrokLabel("error","","<long>") -> len <= package max
```

## Preconditions

Error longer than `menubar` max label length (expected 40).

## Steps

1. Use 80+ character error message.

## Context

REQUIREMENT leaf: `label/error-truncate`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Status = "error"
	req.WeeklyLimit = ""
	req.ErrorMsg = "connection reset by peer while waiting for grok usage API response from daemon"
	return nil
}
```