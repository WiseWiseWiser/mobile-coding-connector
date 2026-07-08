# Scenario

**Feature**: Services menu root when no definitions exist

```
empty service list -> FormatServicesEmptyLabel -> placeholder line
```

## Preconditions

`Op=empty` dispatches to `menubar.FormatServicesEmptyLabel`.

## Steps

1. No inputs required.

## Context

REQUIREMENT section A — empty services dropdown.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "empty"
	return nil
}
```