# Scenario

**Feature**: Delete menu gating and confirm dialog copy

```
status -> CanDeleteCronTask -> bool (false only when running)
name -> FormatDeleteCronConfirm -> `Delete cron task "{name}"?`
```

## Preconditions

`Op=delete-gate` dispatches to `macosapp/menubar` helpers. Mirrors
`CanRunCronTask` pattern (disallow while running).

## Steps

1. Leaf supplies `Status` and/or `Name`.

## Context

REQUIREMENT: Delete disabled when `status == "running"`; confirm dialog
before DELETE.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "delete-gate"
	return nil
}
```
