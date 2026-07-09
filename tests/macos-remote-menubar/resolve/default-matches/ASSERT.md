## Expected

1. `State` is `ok` (config-level: endpoint resolved).
2. `Resolved` is `true`.
3. `Server` is `https://example.com` (no trailing slash).
4. `Token` is `secret`.

## Errors

- Wrong domain selected or token dropped.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.State != "ok" {
		t.Fatalf("state = %q, want ok", resp.State)
	}
	if !resp.Resolved {
		t.Fatal("expected Resolved=true")
	}
	if resp.Server != "https://example.com" {
		t.Fatalf("server = %q, want https://example.com", resp.Server)
	}
	if resp.Token != "secret" {
		t.Fatalf("token = %q, want secret", resp.Token)
	}
}
```
