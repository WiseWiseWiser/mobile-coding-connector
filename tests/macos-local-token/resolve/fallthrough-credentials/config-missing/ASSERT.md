## Expected

1. `Token` is `cred-from-missing-config`.
2. `Source` is `credentials`.

## Errors

- Returning `none` when credentials exist; or erroring on missing config.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.Token != "cred-from-missing-config" {
		t.Fatalf("token = %q, want cred-from-missing-config", resp.Token)
	}
	if resp.Source != "credentials" {
		t.Fatalf("source = %q, want credentials", resp.Source)
	}
}
```
