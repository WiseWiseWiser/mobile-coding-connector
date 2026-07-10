# Scenario

**Feature**: clean project title parts

```
# clean project on main
FormatProjectTitleParts("demo","main",true,"") -> Leading="demo", Trailing="● main"
FormatProjectTitle(...) -> "demo  ● main"
```

## Preconditions

Project is clean with branch `main`; no error.

## Steps

1. Set name `demo`, branch `main`, clean `true`, empty error.

## Context

REQUIREMENT: project clean → Leading `demo`, Trailing `● main`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Name = "demo"
	req.Branch = "main"
	req.Clean = true
	req.ErrMsg = ""
	return nil
}
```
