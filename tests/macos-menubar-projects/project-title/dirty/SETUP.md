# Scenario

**Feature**: dirty project title

```
FormatProjectTitle("demo","main",false,"") -> "demo ○ main"
```

## Preconditions

Project worktree is dirty; branch still `main`.

## Steps

1. Set name `demo`, branch `main`, clean `false`.

## Context

REQUIREMENT leaf: project title dirty.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Name = "demo"
	req.Branch = "main"
	req.Clean = false
	req.ErrMsg = ""
	return nil
}
```
