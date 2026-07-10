## Expected

1. `Line` is exactly `Machine backup — foo.example.com`.

## Errors

- Missing server name, wrong dash (must be em dash `—`), or extra brackets.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "Machine backup — foo.example.com"
	if resp.Line != want {
		t.Fatalf("Line = %q, want %q", resp.Line, want)
	}
}
```
