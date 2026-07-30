# Scenario

**Feature**: service add without --name

```
service add --command "sleep 300" -> non-zero error
```

## Steps

1. CLIArgs omit `--name`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setCLI(req,
		"service", "add",
		"--command", "sleep 300",
	)
	return nil
}
```
