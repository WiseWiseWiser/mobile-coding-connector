# Scenario

**Feature**: empty projects registry label

```
FormatProjectsEmptyLabel() -> "No wrk projects"
```

## Preconditions

GET projects returned an empty list / no registered wrk projects.

## Steps

1. Invoke empty-label formatter (Op already set by parent).

## Context

REQUIREMENT leaf: empty projects label.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	// Reaffirm empty op (parent sets Op; leaf documents the concrete call).
	req.Op = "empty"
	return nil
}
```
