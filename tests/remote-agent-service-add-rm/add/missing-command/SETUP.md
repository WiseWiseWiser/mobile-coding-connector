# Scenario

**Feature**: service add without --command

```
service add --name demo-no-cmd -> non-zero error
```

## Steps

1. CLIArgs omit `--command`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setCLI(req,
		"service", "add",
		"--name", "demo-no-cmd",
	)
	return nil
}
```
