## Expected

1. `GrokDef` and `OpenCodeDef` are non-nil.
2. `GrokDef.Installed` is false.
3. `OpenCodeDef.Installed` is false.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.GrokDef == nil || resp.OpenCodeDef == nil {
		t.Fatal("missing grok or opencode in list")
	}
	if resp.GrokDef.Installed {
		t.Fatal("grok should not be installed without opencode on PATH")
	}
	if resp.OpenCodeDef.Installed {
		t.Fatal("opencode should not be installed on stripped PATH")
	}
}
```