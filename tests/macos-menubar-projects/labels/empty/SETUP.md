# Scenario

**Feature**: empty projects registry label

```
FormatProjectsEmptyLabel() -> "No wrk projects"
```

## Preconditions

Not loading; registry empty; no load error.

## Steps

1. Set `LabelKind=empty`.

## Context

REQUIREMENT: empty label → `No wrk projects`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.LabelKind = "empty"
	return nil
}
```
