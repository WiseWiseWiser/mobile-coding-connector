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
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Mode = "reuse"
	req.UseRealOpenConfig = true
	return nil
}
```
