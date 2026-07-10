## Expected

1. `Token` is `cfg-wins-token`.
2. `Source` is `config` (not `credentials`).

## Errors

- Preferring credentials when config already provides a token.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.Token != "cfg-wins-token" {
		t.Fatalf("token = %q, want cfg-wins-token", resp.Token)
	}
	if resp.Source != "config" {
		t.Fatalf("source = %q, want config", resp.Source)
	}
}
```
