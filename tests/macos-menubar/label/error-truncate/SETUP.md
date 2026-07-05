# Scenario

**Bug**: long grok error must not truncate ugly exec path in menu bar

```
FormatGrokLabel("error","","<long>") -> "Grok err"
```

## Preconditions

Error longer than menu-bar max; label uses fixed short text.

## Steps

1. Use 80+ character error message.

## Context

REQUIREMENT leaf: `label/error-truncate` (amended).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Status = "error"
	req.WeeklyLimit = ""
	req.ErrorMsg = "connection reset by peer while waiting for grok usage API response from daemon"
	return nil
}
```