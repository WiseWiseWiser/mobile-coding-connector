# Scenario

**Feature**: human list on empty tree shows No bookmarks

```
bookmarks list -> stdout contains No bookmarks (or empty root only message)
```

## Preconditions

1. No seeds.

## Steps

1. list without --json.
2. Assert message.

## Context

Empty UX.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.CLIArgs = []string{"bookmarks", "list"}
	return nil
}
```
