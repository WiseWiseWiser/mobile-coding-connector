# Scenario

**Feature**: reject unexpected positional arguments

```
# paste-bin foo -> usage/args error before API side effects
extra positional arg -> non-zero exit
```

## Preconditions

Any scratch state; error occurs at CLI parse layer.

## Steps

1. `seedScratch(req, seededUTF8Content, "")` — incidental seed.
2. `req.Args = []string{"paste-bin", "foo"}`; TTY stdin.

## Context

REQUIREMENT leaf: `reject-extra-args`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	seedScratch(req, seededUTF8Content, "")
	req.Args = []string{"paste-bin", "foo"}
	req.PipedStdin = nil
	req.StdinBytes = nil
	return nil
}
```