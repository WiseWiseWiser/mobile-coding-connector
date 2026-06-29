# Scenario

**Feature**: bootstrap core_ready timestamp precedes extension_start

```
[bootstrap] phase=core_ready t_ms=A
[bootstrap] phase=extension_start t_ms=B   (require A < B)
```

## Preconditions

5s extension delay.

## Steps

1. `ExtensionDelayMs=5000`, `ObserveSecs=10`.

## Context

Requires structured bootstrap timing logs from the implementer.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ExtensionDelayMs = 5000
	req.ObserveSecs = 10
	return nil
}
```