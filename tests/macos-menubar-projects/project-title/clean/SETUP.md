# Scenario

**Feature**: clean project title

```
FormatProjectTitle("demo","main",true,"") -> "demo ● main"
```

## Preconditions

Project is clean with branch `main`.

## Steps

1. Set name `demo`, branch `main`, clean `true`, empty error.

## Context

REQUIREMENT leaf: project title clean.

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
