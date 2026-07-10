## Expected

1. `Token` is empty.
2. `Source` is `none`.

## Errors

- Returning whitespace as token; treating blank file as credentials source with empty string inconsistently (must be `none`).

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
