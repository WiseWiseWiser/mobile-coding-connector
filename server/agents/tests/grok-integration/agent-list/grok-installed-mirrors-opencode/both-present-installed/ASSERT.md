## Expected

1. `GrokDef.Installed` is true.
2. `OpenCodeDef.Installed` is true.
3. `GrokDef.Installed == OpenCodeDef.Installed`.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.GrokDef == nil || resp.OpenCodeDef == nil {
		t.Fatal("missing grok or opencode in list")
	}
	if !resp.GrokDef.Installed {
		t.Fatal("grok should be installed when fake opencode is on PATH")
	}
	if !resp.OpenCodeDef.Installed {
		t.Fatal("opencode should be installed when fake opencode is on PATH")
	}
	if resp.GrokDef.Installed != resp.OpenCodeDef.Installed {
		t.Fatal("grok and opencode installed flags must match")
	}
}
```