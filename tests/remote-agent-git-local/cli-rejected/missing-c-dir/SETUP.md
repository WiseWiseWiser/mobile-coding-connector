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

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Args = []string{"git", "status"}
	return nil
}
```