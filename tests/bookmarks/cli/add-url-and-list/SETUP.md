# Scenario

**Feature**: bookmarks add creates url visible in list

```
bookmarks add --name N --url U --id bm_cli -> list shows N
```

## Preconditions

1. add then rely on Doc reload / list not required if Doc updated.

## Steps

1. add with fixed id.
2. Assert Doc or subsequent state has node.

## Context

CLI create.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.CLIArgs = []string{
		"bookmarks", "add",
		"--name", "CLI Dash",
		"--url", "http://127.0.0.1:7070",
		"--id", "bm_cli",
	}
	return nil
}
```
