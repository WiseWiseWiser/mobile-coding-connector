# Scenario

**Feature**: default services list is project-scoped

```
GET /api/services -> only services matching server project dir
```

## Preconditions

Two services seeded: one local project, one other project.

## Steps

1. Set `Op=list-scoped`.

## Context

REQUIREMENT leaf: `list-scoped-default`.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "list-scoped"
	return nil
}
```