# Scenario

**Feature**: project error title

```
FormatProjectTitle("demo",...,errMsg!=\"\") -> "demo ⚠ Error"
```

## Preconditions

Project has a non-empty error (e.g. missing path); error presentation takes
priority over clean/branch.

## Steps

1. Set name `demo`, any branch, `ErrMsg` non-empty.

## Context

REQUIREMENT leaf: project error.

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
