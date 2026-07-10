## Expected

1. `Token` is `cred-after-bad-json`.
2. `Source` is `credentials`.
3. Resolve does not return a fatal error for bad config JSON.

## Errors

- Hard-failing resolve on invalid config; skipping credentials.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.Token != "cred-after-bad-json" {
		t.Fatalf("token = %q, want cred-after-bad-json", resp.Token)
	}
	if resp.Source != "credentials" {
		t.Fatalf("source = %q, want credentials", resp.Source)
	}
}
```
