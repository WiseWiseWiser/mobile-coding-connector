# Scenario

**Feature**: all enabled Print flags produce expected stdout

```
# log -> "  hello log\n"; section -> "Server checks:\n"; progress -> "[ok] ..."
```

## Preconditions

Default mock events from root `Run`.

## Steps

No additional setup.

## Context

Validates A-path without custom `Printer`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.MockEvents = defaultStreamcmdEvents()
	return nil
}
```
