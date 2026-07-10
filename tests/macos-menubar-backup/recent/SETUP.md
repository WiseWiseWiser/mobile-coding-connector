# Scenario

**Feature**: recent backup list empty label, sort, and entry format

```
entries + now -> empty label | sorted paths | "rel · size"
```

## Preconditions

Pure entry structs (no filesystem required for these leaves).

## Steps

1. Leaf sets `Op` to `recent_empty`, `recent_list`, or `recent_format`.

## Context

REQUIREMENT recent list scenarios 14–16.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	if req.NowRFC3339 == "" {
		req.NowRFC3339 = defaultNowRFC3339
	}
	return nil
}
```
