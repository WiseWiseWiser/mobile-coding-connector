# Scenario

**Feature**: Projects menu placeholder label strings (empty / loading / failed)

```
# pure label constants
() -> FormatProjectsEmptyLabel -> "No wrk projects"
() -> FormatProjectsLoadingLabel -> "Loading…"
() -> FormatProjectsLoadFailedLabel -> "Failed to load projects"
```

## Preconditions

`Op=label` dispatches by `LabelKind` to the matching formatter.

## Steps

1. Set `Op=label`.
2. Leaf sets `LabelKind` to `empty`, `loading`, or `failed`.

## Context

REQUIREMENT scenarios 5–7.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "label"
	return nil
}
```
