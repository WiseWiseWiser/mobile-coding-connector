# Scenario

**Feature**: parse /status usage fields from signed post-Skip status screen

```
05-status-fields.snapshot.txt -> ParseStatusSnapshot monthly/credits/reset
```

## Preconditions

Signed fixture after successful Skip + `/status` on live Codex 0.143.0.

## Steps

1. Leaf sets fixture `05-status-fields.snapshot.txt`.

## Context

PROTOCOL step `continue_status`. Ensures field regexes still match real TUI chrome.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	t.Helper()
	if req.Op == "" {
		req.Op = "classify"
	}
	return nil
}
```
