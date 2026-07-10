## Expected

1. `BuildOK` is true.
2. `HasAuth` is false.
3. `AuthHeader` is empty.

## Errors

- Spurious `Bearer ` header when token is empty.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.BuildOK {
		t.Fatalf("build failed: %s", resp.BuildErr)
	}
	if resp.HasAuth {
		t.Fatalf("unexpected Authorization: %q", resp.AuthHeader)
	}
	if resp.AuthHeader != "" {
		t.Fatalf("auth header = %q, want empty", resp.AuthHeader)
	}
}
```
