## Expected

1. `ModelSubstring` is `"grok"` (case-sensitive per requirement substring match).
2. `ModelSubstring` is not `"kimi-k2.5"`.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ModelSubstring != "grok" {
		t.Fatalf("ModelSubstring = %q, want grok", resp.ModelSubstring)
	}
	if resp.ModelSubstring == "kimi-k2.5" {
		t.Fatal("grok must not use default kimi preference")
	}
}
```