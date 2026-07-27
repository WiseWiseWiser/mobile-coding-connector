# Scenario

**Feature**: empty services list placeholder

```
FormatServicesEmptyLabel() -> "No services configured"
```

## Preconditions

`GET /api/services?all=1` returned an empty array.

## Steps

1. Invoke empty-label formatter.

## Context

REQUIREMENT leaf: `dropdown/empty`.

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