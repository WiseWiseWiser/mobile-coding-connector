## Expected

1. `State` is `ok`.
2. `Resolved` is `true`.
3. `Server` is `https://x.com`.
4. `Token` is `ws-tok`.

## Errors

- Whitespace prevents match.

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
		t.Fatalf("server = %q, want https://x.com", resp.Server)
	}
	if resp.Token != "ws-tok" {
		t.Fatalf("token = %q, want ws-tok", resp.Token)
	}
}
```
