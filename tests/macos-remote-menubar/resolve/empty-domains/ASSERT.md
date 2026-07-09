## Expected

1. `State` is `not_configured`.
2. `Resolved` is `false`.

## Errors

- Falling back to a hard-coded server when domains are empty.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.State != "not_configured" {
		t.Fatalf("state = %q, want not_configured", resp.State)
	}
	if resp.Resolved {
		t.Fatal("expected Resolved=false for empty domains")
	}
}
```
