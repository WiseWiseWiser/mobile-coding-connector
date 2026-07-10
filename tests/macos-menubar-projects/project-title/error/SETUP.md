# Scenario

**Feature**: project error title parts

```
# error wins over clean/branch
FormatProjectTitleParts("demo",...,errMsg!="") -> Leading="demo", Trailing="⚠ Error"
FormatProjectTitle(...) -> "demo  ⚠ Error"
```

## Preconditions

Project has a non-empty error (e.g. missing path); error presentation takes
priority over clean/branch.

## Steps

1. Set name `demo`, any branch, `ErrMsg` non-empty.

## Context

REQUIREMENT: project error → Leading `demo`, Trailing `⚠ Error`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Name = "demo"
	req.Branch = "main"
	req.Clean = false
	req.ErrMsg = "path does not exist"
	return nil
}
```
