## Expected

1. `Token` is `local-host-token` (localhost domain), not `other-token`.
2. `Source` is `config`.

## Errors

- Using the default-domain token when a local loopback domain exists.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.Token != "local-host-token" {
		t.Fatalf("token = %q, want local-host-token", resp.Token)
	}
	if resp.Source != "config" {
		t.Fatalf("source = %q, want config", resp.Source)
	}
}
```
