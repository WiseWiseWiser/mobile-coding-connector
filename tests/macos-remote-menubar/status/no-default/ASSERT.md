## Expected

1. `StatusLine` is exactly:
   `Multiple servers configured — open Configure… to pick a default`
2. `StatusContainsConfig` is true.
3. `StatusContainsToken` is false.

## Errors

- Silent status or auto-picking a domain without guidance.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := statusNoDefault
	if resp.StatusLine != want {
		t.Fatalf("StatusLine = %q, want %q", resp.StatusLine, want)
	}
	if !resp.StatusContainsConfig {
		t.Fatal("expected StatusLine to mention Configure")
	}
	if resp.StatusContainsToken {
		t.Fatalf("status leaked token: %q", resp.StatusLine)
	}
}
```
