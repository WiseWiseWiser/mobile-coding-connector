## Expected

1. `State` is `ok`.
2. `Resolved` is `true`.
3. `Server` is `https://only.example`.
4. `Token` is `only-tok`.

## Errors

- Returning `no_default` when a single domain is present.

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
	if resp.Server != "https://only.example" {
		t.Fatalf("server = %q, want https://only.example", resp.Server)
	}
	if resp.Token != "only-tok" {
		t.Fatalf("token = %q, want only-tok", resp.Token)
	}
}
```
