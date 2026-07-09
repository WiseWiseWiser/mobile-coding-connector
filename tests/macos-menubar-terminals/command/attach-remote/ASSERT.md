## Expected

1. `Command` is exactly `remote-agent terminal attach web1`.

## Errors

- Embedding Bearer token, wrong binary, or attach by display name only.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "remote-agent terminal attach web1"
	if resp.Command != want {
		t.Fatalf("command = %q, want %q", resp.Command, want)
	}
}
```
