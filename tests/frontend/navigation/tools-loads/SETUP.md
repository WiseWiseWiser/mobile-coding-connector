# Scenario

**Feature**: Server Tools page loads

```
# user navigates to /home/tools
Playwright -> BASE_URL/home/tools -> Server Tools heading or Foundation category
```

## Preconditions

1. Quick-test server is running and healthy.
2. The frontend route `/home/tools` is registered in the v2 router.

## Steps

1. Set `Request.ScriptPath` to `script.js`.
2. Set `Request.TimeoutSecs` to `120` to allow the tools SSE stream to complete.
3. The fixture navigates to `BASE_URL + '/home/tools'`.
4. Wait for `h2` heading or the `Foundation` category label.

## Context

The tools page streams installation status via SSE. The test accepts either the
`Server Tools` page heading or a visible `Foundation` category section as proof
the page loaded.

```go
import (
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	req.ScriptPath = "script.js"
	req.TimeoutSecs = 120
	return nil
}
```