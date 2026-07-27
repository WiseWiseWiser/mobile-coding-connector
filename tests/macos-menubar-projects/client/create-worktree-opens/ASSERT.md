## Expected

1. `CreateWorktreeOpensThenRefresh` is true — create success path opens the
   returned directory and calls `refreshProjects`.

## Errors

- Create then refresh only, without open (pre-impl).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.CreateWorktreeOpensThenRefresh {
		t.Fatalf("create worktree must open returned path then refresh (sources: %v)", resp.SwiftSourcesChecked)
	}
}
```
