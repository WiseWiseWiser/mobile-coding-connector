# Scenario

**Feature**: home workspace list page loads

```
# user navigates to /home
Playwright -> BASE_URL/home -> WorkspaceListView (.mcc-workspace-list)
```

## Preconditions

1. Quick-test server is running and healthy.
2. The frontend route `/home` is registered in the v2 router.

## Steps

1. Set `Request.ScriptPath` to `script.js`.
2. The fixture navigates to `BASE_URL + '/home'`.
3. Wait for `.mcc-workspace-list` and verify the workspace UI is present.

## Context

The home page renders `WorkspaceListView` with a `Your Projects` heading and
workspace list container. An empty project list is acceptable; the page must
still render its shell UI.

```go
import (
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	req.ScriptPath = "script.js"
	if req.TimeoutSecs <= 0 {
		req.TimeoutSecs = 90
	}
	return nil
}
```