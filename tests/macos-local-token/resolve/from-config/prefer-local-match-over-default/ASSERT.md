## Expected

1. `Token` is `prefer-local-token` (not `default-domain-token`).
2. `Source` is `config`.
3. Trailing slash on domain server does not prevent local match.

## Errors

- Returning the default-domain token when a usable local domain exists.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.Token != "prefer-local-token" {
		t.Fatalf("token = %q, want prefer-local-token", resp.Token)
	}
	if resp.Source != "config" {
		t.Fatalf("source = %q, want config", resp.Source)
	}
}
```
