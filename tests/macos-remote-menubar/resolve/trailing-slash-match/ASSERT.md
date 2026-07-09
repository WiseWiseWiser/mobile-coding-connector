## Expected

1. `State` is `ok`.
2. `Resolved` is `true`.
3. `Server` is `https://x.com` (no trailing slash).
4. `Token` is `tok-x`.

## Errors

- Failing to match because of trailing slash.

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
	if resp.Server != "https://x.com" {
		t.Fatalf("server = %q, want https://x.com (no trailing slash)", resp.Server)
	}
	if resp.Token != "tok-x" {
		t.Fatalf("token = %q, want tok-x", resp.Token)
	}
}
```
