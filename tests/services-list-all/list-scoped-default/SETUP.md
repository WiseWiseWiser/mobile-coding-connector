# Scenario

**Feature**: default services list returns every service

```
Manager.List() -> local-web + other-api
```

## Preconditions

Two services seeded.

## Steps

1. Set `Op=list`.

## Context

REQUIREMENT leaf: `list-scoped-default` (legacy name; list is global).

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "list"
	return nil
}
```
