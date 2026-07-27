# Scenario

**Feature**: error service title

```
FormatServiceTitle("web","error",...) -> "web ⚠ Error"
```

## Preconditions

Service status is `error`; title includes the full service name and error presentation.

## Steps

1. Set status `error` with multi-character name `web`.

## Context

REQUIREMENT leaf: `title/error`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Name = "web"
	req.Status = "error"
	req.Enabled = true
	return nil
}
```
