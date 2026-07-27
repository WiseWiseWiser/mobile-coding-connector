# Scenario

**Feature**: per-project submenu title parts (Leading left / Trailing right)

```
# name + branch + clean/error -> FormatProjectTitleParts -> {Leading, Trailing}
name, branch, clean, errMsg -> FormatProjectTitleParts -> Leading, Trailing
legacy FormatProjectTitle -> Leading + "  " + Trailing
```

## Preconditions

`Op=project_title` dispatches to `menubar.FormatProjectTitleParts` and legacy
`FormatProjectTitle`.

## Steps

1. Leaf supplies `Name`, `Branch`, `Clean`, and optional `ErrMsg`.

## Context

REQUIREMENT scenarios 1–3 (project clean / dirty / error).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "project_title"
	return nil
}
```
