# Scenario

**Feature**: grok usage service fetch via tty library and mock command

```
GROK_SHOW_USAGE_COMMAND fake TUI -> tty.FetchUsageWithOptions -> service FetchOnce -> GrokUsageResponse
```

## Preconditions

Mock fake-TUI scripts in `testdata/`; `TestExported_SetEnv` sets `GROK_SHOW_USAGE_COMMAND`.

## Steps

1. Set `Op=fetch` in leaves.

## Context

Service-layer tests without full daemon HTTP; no `GROK_SHOW_USAGE_BIN` exec.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "fetch"
	return nil
}
```