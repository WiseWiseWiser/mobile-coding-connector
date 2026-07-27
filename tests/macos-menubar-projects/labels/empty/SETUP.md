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
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.LabelKind = "empty"
	return nil
}
```
