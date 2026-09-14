# Scenario

**Feature**: ListAll returns every service

```
Manager.ListAll() -> local-web + other-api
```

## Preconditions

Two services seeded.

## Steps

1. Set `Op=list-all`.

## Context

REQUIREMENT leaf: `list-all`.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "list-all"
	return nil
}
```
