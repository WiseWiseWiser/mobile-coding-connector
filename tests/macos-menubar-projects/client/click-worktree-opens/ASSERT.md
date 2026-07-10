## Expected

1. `ClickWorktreeOpensPath` is true.

## Errors

- Display-only worktree Button with empty action (current pre-impl state).

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.ClickWorktreeOpensPath {
		t.Fatalf("worktree click must open worktree path via openITerm2 (sources: %v)", resp.SwiftSourcesChecked)
	}
}
```
