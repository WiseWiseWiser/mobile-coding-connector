# Scenario

**Feature**: event-bus help surfaces

```
remote-agent [--help] | event-bus --help | event-bus listen --help
  -> Usage text on stdout
```

## Steps

1. Op=cli; leaf sets Args for the help depth under test.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	req.Op = "cli"
	return nil
}
```
