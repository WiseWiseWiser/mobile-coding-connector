# Scenario

**Feature**: sealed 30-second periodic refresh interval

```
PeriodicRefreshInterval == 30 * time.Second
```

## Preconditions

Go constant (or duration helper) documents the app-side poll period.

## Steps

1. Run with `Op=interval`.

## Context

REQUIREMENT leaf: periodic refresh interval constant.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "interval"
	return nil
}
```
