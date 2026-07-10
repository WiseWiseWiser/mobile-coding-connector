## Expected

1. `Token` is `first-non-empty-cred` (not blank, not second line).
2. `Source` is `credentials`.

## Errors

- Returning empty because first physical lines are blank; or using the second token.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.Token != "first-non-empty-cred" {
		t.Fatalf("token = %q, want first-non-empty-cred", resp.Token)
	}
	if resp.Source != "credentials" {
		t.Fatalf("source = %q, want credentials", resp.Source)
	}
}
```
