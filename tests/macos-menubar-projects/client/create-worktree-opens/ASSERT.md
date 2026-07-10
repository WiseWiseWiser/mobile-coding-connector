## Expected

1. `CreateWorktreeOpensThenRefresh` is true — create success path opens the
   returned directory and calls `refreshProjects`.

## Errors

- Create then refresh only, without open (pre-impl).

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.CreateWorktreeOpensThenRefresh {
		t.Fatalf("create worktree must open returned path then refresh (sources: %v)", resp.SwiftSourcesChecked)
	}
}
```
