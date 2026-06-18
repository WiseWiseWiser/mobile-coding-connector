# Scenario

**Feature**: Settings page loads

```
# user navigates to /home/settings
Playwright -> BASE_URL/home/settings -> SettingsView h2
```

## Preconditions

1. Quick-test server is running and healthy.
2. The frontend route `/home/settings` is registered in the v2 router.

## Steps

1. Set `Request.ScriptPath` to `script.js`.
2. The fixture navigates to `BASE_URL + '/home/settings'`.
3. Wait for the `Settings` heading in `h2`.

## Context

The settings page renders `SettingsView` with a section header containing an
`h2` with text `Settings`.

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