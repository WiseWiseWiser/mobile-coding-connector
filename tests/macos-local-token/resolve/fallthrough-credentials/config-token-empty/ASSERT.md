## Expected

1. `Token` is `cred-after-empty-cfg` (not whitespace).
2. `Source` is `credentials`.

## Errors

- Treating whitespace as a usable config token; or returning raw spaces.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.Token != "cred-after-empty-cfg" {
		t.Fatalf("token = %q, want cred-after-empty-cfg", resp.Token)
	}
	if resp.Source != "credentials" {
		t.Fatalf("source = %q, want credentials", resp.Source)
	}
}
```
