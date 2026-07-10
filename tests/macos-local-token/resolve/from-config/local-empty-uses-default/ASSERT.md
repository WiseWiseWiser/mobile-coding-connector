## Expected

1. `Token` is `from-default-after-local-empty`.
2. `Source` is `config` (default domain used after empty local match).

## Errors

- Skipping default domain and going to credentials when default has a token.
- Returning empty because localhost domain matched with empty token.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.Token != "from-default-after-local-empty" {
		t.Fatalf("token = %q, want from-default-after-local-empty", resp.Token)
	}
	if resp.Source != "config" {
		t.Fatalf("source = %q, want config", resp.Source)
	}
}
```
