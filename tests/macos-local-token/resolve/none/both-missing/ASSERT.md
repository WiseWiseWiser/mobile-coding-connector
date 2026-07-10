## Expected

1. `Token` is empty string.
2. `Source` is `none`.
3. No fatal error from resolve.

## Errors

- Returning a non-empty token; hard error on missing files.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.Token != "" {
		t.Fatalf("token = %q, want empty", resp.Token)
	}
	if resp.Source != "none" {
		t.Fatalf("source = %q, want none", resp.Source)
	}
}
```
