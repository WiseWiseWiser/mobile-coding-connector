# Scenario

**Feature**: mode=reuse selects ModeReuseCurrent

```
POST {dir, mode:"reuse"} -> ModeReuseCurrent -> 200
```

## Preconditions

Valid temp dir.

## Steps

1. Set `Mode=reuse`.

## Context

REQUIREMENT mode map reuse.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Mode = "reuse"
	req.UseRealOpenConfig = true
	return nil
}
```
