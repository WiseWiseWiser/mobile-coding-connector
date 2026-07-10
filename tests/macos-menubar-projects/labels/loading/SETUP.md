# Scenario

**Feature**: projects list loading label

```
FormatProjectsLoadingLabel() -> "Loading…"
```

## Preconditions

Projects list request is in flight; UI needs a loading placeholder (especially
when the prior list is empty).

## Steps

1. Set `LabelKind=loading`.

## Context

REQUIREMENT: loading label → `Loading…` (unicode ellipsis U+2026, not three dots).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.LabelKind = "loading"
	return nil
}
```
