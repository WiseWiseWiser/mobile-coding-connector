# Scenario

**Feature**: Doctor fails serve check when ServeOK returns error

```
ServeOK -> error "connection refused"
  -> Doctor -> serve OK=false, AllOK false
```

## Preconditions

- Seeded pair + profile; versions match.

## Steps

1. Override ServeOK to fail.

## Context

- Remote-root may also fail when serve is down; only `serve` is required fail.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.ServeOK = serveDown("connection refused")
	return nil
}
```
