## Expected

1. `Command` is exactly `local-agent terminal attach web1`.

## Errors

- Using session name instead of id, wrong binary, or extra flags embedding tokens.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "local-agent terminal attach web1"
	if resp.Command != want {
		t.Fatalf("command = %q, want %q", resp.Command, want)
	}
}
```
