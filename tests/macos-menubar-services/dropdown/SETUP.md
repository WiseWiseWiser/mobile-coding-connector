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
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "empty"
	return nil
}
```