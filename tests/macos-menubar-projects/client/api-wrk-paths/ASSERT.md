## Expected

1. `HasAPIWrkProjects` is true — sources reference `/api/wrk/projects`.
2. `HasAPIWrkWorktrees` is true — sources reference `/api/wrk/worktrees`.
3. `ServerPort` equals default server port (`23712`).

## Side Effects

- None (read-only source inspection).

## Errors

- Paths hardcoded under a different prefix, or missing create endpoint.

```go
import (
	"testing"

	"github.com/xhd2015/ai-critic/server/config"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.ServerPort != config.DefaultServerPort {
		t.Fatalf("ServerPort = %d, want %d", resp.ServerPort, config.DefaultServerPort)
	}
	if !resp.HasAPIWrkProjects {
		t.Fatalf("missing /api/wrk/projects in sources: %v", resp.SwiftSourcesChecked)
	}
	if !resp.HasAPIWrkWorktrees {
		t.Fatalf("missing /api/wrk/worktrees in sources: %v", resp.SwiftSourcesChecked)
	}
}
```
