# Scenario

**Feature**: mock fetch failure → no invented structured reset fields

```
GROK_SHOW_USAGE_COMMAND=mock-fail.sh -> tty fetch error -> status=error
  reset_at, reset_display, time_left empty
```

## Preconditions

`mock-fail.sh` exits non-zero after prompt so the service records `status=error`.

## Steps

1. Set `MockScript=mock-fail.sh`.

## Context

REQUIREMENT-DESIGN-usage-structured-reset-ab.md scenario 3: on error do not invent
`reset_at` / `reset_display` / `time_left`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.MockScript = "mock-fail.sh"
	return nil
}
```
