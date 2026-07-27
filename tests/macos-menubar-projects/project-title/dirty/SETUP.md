# Scenario

**Feature**: dirty project title parts

```
# dirty project on main
FormatProjectTitleParts("demo","main",false,"") -> Leading="demo", Trailing="○ main"
FormatProjectTitle(...) -> "demo  ○ main"
```

## Preconditions

Project worktree is dirty; branch still `main`; no error.

## Steps

1. Set name `demo`, branch `main`, clean `false`.

## Context

REQUIREMENT: project dirty → Leading `demo`, Trailing `○ main`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Name = "demo"
	req.Branch = "main"
	req.Clean = false
	req.ErrMsg = ""
	return nil
}
```
