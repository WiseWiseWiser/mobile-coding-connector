## Expected

1. `Command` is exactly `remote-agent terminal new`.

## Errors

- Using local-agent from remote app, or embedding tokens.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "remote-agent terminal new"
	if resp.Command != want {
		t.Fatalf("command = %q, want %q", resp.Command, want)
	}
}
```
