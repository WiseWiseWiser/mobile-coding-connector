## Expected

1. `State` is `no_default`.
2. `Resolved` is `false`.
3. `Server` and `Token` are empty (must not silently pick first domain).

## Errors

- Auto-selecting first domain when product requires Configure… for multi-domain.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.State != "no_default" {
		t.Fatalf("state = %q, want no_default", resp.State)
	}
	if resp.Resolved {
		t.Fatal("expected Resolved=false for multi-domain without default")
	}
	if resp.Server != "" || resp.Token != "" {
		t.Fatalf("endpoint = {%q, %q}, want empty (no silent first-domain pick)", resp.Server, resp.Token)
	}
}
```
