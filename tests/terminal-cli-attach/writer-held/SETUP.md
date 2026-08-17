# Scenario

**Feature**: another client already holds the exclusive writer

## Steps

1. Create a live shell via REST (no writer yet).
2. Open `/api/terminal?session_id=…` with default attach mode so the harness
   claims `writer` and keeps the socket open.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	if req.Phase == "" {
		req.Phase = "writer-held-echo"
	}
	return nil
}
```
