# Scenario

**Feature**: local-agent --show --web mutual exclusion

```
local-agent config --show --web -> non-zero error
```

## Preconditions

None.

## Steps

1. Args = `config --show --web`.

## Context

Parity T7.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Args = []string{"config", "--show", "--web"}
	return nil
}
```
