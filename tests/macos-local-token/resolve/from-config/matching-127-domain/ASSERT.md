## Expected

1. `Token` is `local-127-token`.
2. `Source` is `config`.

## Errors

- Treating only `localhost` as local and ignoring `127.0.0.1`.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.Token != "local-127-token" {
		t.Fatalf("token = %q, want local-127-token", resp.Token)
	}
	if resp.Source != "config" {
		t.Fatalf("source = %q, want config", resp.Source)
	}
}
```
