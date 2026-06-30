## Expected

1. `LaunchErr` is nil (list path only).
2. `GrokDef` is non-nil.
3. `GrokDef.ID` is `"grok"`.
4. `GrokDef.Name` is `"Grok"`.
5. `GrokDef.Command` is `"opencode"`.
6. `GrokDef.Headless` is true.
7. `len(ListAgents)` is at least 5.

## Errors

- Missing grok entry or wrong headless/command fails the test.

```go
import (
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.GrokDef == nil {
		t.Fatalf("grok not in agent list: %s", resp.ListBody)
	}
	g := resp.GrokDef
	if g.ID != "grok" {
		t.Fatalf("id = %q", g.ID)
	}
	if g.Name != "Grok" {
		t.Fatalf("name = %q", g.Name)
	}
	if g.Command != "opencode" {
		t.Fatalf("command = %q", g.Command)
	}
	if !g.Headless {
		t.Fatal("expected headless true")
	}
	if len(resp.ListAgents) < 5 {
		t.Fatalf("expected at least 5 agents, got %d", len(resp.ListAgents))
	}
}
```