## Expected

1. `Command` is exactly `local-agent terminal attach web1`.

## Errors

- Using session name instead of id, wrong binary, or extra flags embedding tokens.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "local-agent terminal attach web1"
	if resp.Command != want {
		t.Fatalf("command = %q, want %q", resp.Command, want)
	}
}
```
