## Expected

1. `StatusLine` is exactly:
   `Not configured — open Configure… to add a remote server`
2. `StatusContainsConfig` is true.
3. `StatusContainsToken` is false.

## Errors

- Silent empty status; mentioning raw tokens; telling user to copy CLI config only.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := statusNotConfigured
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
