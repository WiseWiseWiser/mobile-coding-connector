# Scenario

**Feature**: HTTP /ping succeeds before extension auto-task logs

```
/ping OK -> later "[auto-task] Running extension" or extension_start marker
```

## Preconditions

5s extension delay, extension config armed.

## Steps

1. `ExtensionDelayMs=5000`, `ObserveSecs=10`.

## Context

Proves port is live before extension work begins (user-visible invariant).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ExtensionDelayMs = 5000
	req.ObserveSecs = 10
	return nil
}
```