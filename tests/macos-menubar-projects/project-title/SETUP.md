# Scenario

**Feature**: per-project submenu title strings

```
name + branch + clean/error -> FormatProjectTitle -> title line
```

## Preconditions

`Op=project_title` dispatches to `menubar.FormatProjectTitle`.

## Steps

1. Leaf supplies `Name`, `Branch`, `Clean`, and optional `ErrMsg`.

## Context

REQUIREMENT scenarios 12–14.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "project_title"
	return nil
}
```
