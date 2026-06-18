## Preconditions

1. Quick-test server is running and healthy.
2. The v2 router redirects `/` to `/home` via `<Navigate to="home" replace />`.

## Steps

1. Set `Request.ScriptPath` to `script.js`.
2. The fixture navigates to `BASE_URL + '/'`.
3. Wait for the browser URL to include `/home`.

## Context

The root route is a redirect-only index route. The test verifies client-side
navigation lands on `/home` without requiring workspace UI to render.

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