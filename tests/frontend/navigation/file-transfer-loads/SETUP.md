# Scenario

**Feature**: File Transfer page loads from direct navigation

```
# user opens /home/file-transfer
Playwright -> BASE_URL/home/file-transfer -> FileTransferView (heading + upload area)
```

## Preconditions

1. Quick-test server is running and healthy.
2. The frontend route `/home/file-transfer` is registered in the v2 router.
3. `FileTransferView` renders a page heading and an upload area.

## Steps

1. Set `Request.ScriptPath` to `script.js`.
2. The fixture navigates to `BASE_URL + '/home/file-transfer'`.
3. Wait for the `File Transfer` heading and verify the upload area is visible.

## Context

This leaf verifies the dedicated File Transfer inbox page shell without
requiring files in storage. The upload area may be a `data-testid`, CSS class,
or upload button — the script accepts any of these signals.

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