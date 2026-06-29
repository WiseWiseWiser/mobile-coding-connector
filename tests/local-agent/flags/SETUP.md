# Scenario

**Feature**: global `--server` and `--port` flag behavior

```
# parse globals before subcommand; port shorthand or conflict
local-agent --port/--server -> URL resolution -> (error | API call)
```

## Preconditions

Flag leaves exercise parsing and resolution before or during the first API call.

## Steps

1. Child `Setup` sets `Port`, `Server`, and subcommand `Args` for each leaf.

## Context

Split factor: mutual exclusivity vs valid `--port` shorthand.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	if req.Args == nil {
		req.Args = []string{}
	}
	req.GlobalHelp = false
	return nil
}
```