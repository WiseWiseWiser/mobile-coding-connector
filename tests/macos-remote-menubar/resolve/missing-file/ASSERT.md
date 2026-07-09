## Expected

1. `State` is `not_configured`.
2. `Resolved` is `false`.
3. `Server` and `Token` are empty.

## Errors

- Treating missing file as fatal error instead of empty config.
- Returning a default localhost endpoint.

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
		t.Fatal("expected Resolved=false for missing config file")
	}
	if resp.Server != "" || resp.Token != "" {
		t.Fatalf("endpoint = {%q, %q}, want empty", resp.Server, resp.Token)
	}
}
```
