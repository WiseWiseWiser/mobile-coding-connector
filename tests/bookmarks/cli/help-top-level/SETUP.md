# Scenario

**Feature**: bookmarks help lists subcommands

```
bookmarks -h -> usage includes list add set delete move open
```

## Preconditions

1. CLIArgs bookmarks -h (or --help).

## Steps

1. Run help.
2. Assert stdout/stderr documents key subcommands.

## Context

Discoverability.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.CLIArgs = []string{"bookmarks", "-h"}
	return nil
}
```
