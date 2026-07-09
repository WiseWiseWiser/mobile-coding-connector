## Expected

1. `Command` is exactly `local-agent terminal new`.

## Errors

- Adding cwd prompt flags or wrong binary name.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "local-agent terminal new"
	if resp.Command != want {
		t.Fatalf("command = %q, want %q", resp.Command, want)
	}
}
```
