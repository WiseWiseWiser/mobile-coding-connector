# Scenario

**Feature**: bookmarks add-folder creates folder node

```
bookmarks add-folder --name Work --id fld_work
```

## Preconditions

1. add-folder subcommand.

## Steps

1. Create folder Work.
2. Assert type folder in Doc.

## Context

CLI folder create.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.CLIArgs = []string{
		"bookmarks", "add-folder",
		"--name", "Work",
		"--id", "fld_work",
	}
	return nil
}
```
