# Scenario

**Feature**: local-agent --json without --show

```
local-agent config --json -> non-zero, mentions --show
```

## Preconditions

None.

## Steps

1. Args = `config --json`.

## Context

Parity T6.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Args = []string{"config", "--json"}
	return nil
}
```
