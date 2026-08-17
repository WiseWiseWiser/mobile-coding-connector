# Scenario

**Feature**: attach must not open a mute TTY on an exited session

## Steps

1. Create a shell, claim writer, close the writer so the child is reaped.
2. Leaf asserts the CLI gate rejects attach.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	if req.Phase == "" {
		req.Phase = "exited-refuse"
	}
	return nil
}
```
