## Expected

1. `CanStop` is `false`.

## Errors

- Stop remains enabled when pid=0 and desiredRunning=false.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.CanStop {
		t.Fatal("CanStop = true, want false when pid=0 && !desiredRunning")
	}
}
```