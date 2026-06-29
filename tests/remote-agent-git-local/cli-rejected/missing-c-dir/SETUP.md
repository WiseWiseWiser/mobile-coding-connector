# Scenario

**Feature**: repo-scoped git subcommands require `-C <dir>`

```
remote-agent git status -> CLI error (requires -C), no HTTP
```

## Preconditions

None.

## Steps

1. Set `Request.Args` to `git status` without `-C`.

## Context

REQUIREMENT leaf #9.

```go
import (
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	req.Args = []string{"git", "status"}
	return nil
}
```