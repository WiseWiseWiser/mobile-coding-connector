# Scenario

**Feature**: empty Projects list placeholder

```
FormatProjectsEmptyLabel() -> "No wrk projects"
```

## Preconditions

`Op=empty` dispatches to `menubar.FormatProjectsEmptyLabel`.

## Steps

1. Set `Op=empty`.

## Context

REQUIREMENT scenario 15.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "empty"
	return nil
}
```
