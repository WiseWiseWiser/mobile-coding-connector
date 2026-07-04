# Scenario

**Feature**: port-in-use error when `--kill-existing` absent

```
occupied server port -> keep-alive without flag -> error, no management API
```

## Preconditions

Parent `no-flag` setup.

## Steps

1. Poll for process exit within `StartupWaitSecs`.

## Context

REQUIREMENT leaf: `no-flag/port-occupied-errors`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.KillExisting = false
	req.OccupyServerPort = true
	req.ExpectStart = false
	req.ExpectError = true
	if req.StartupWaitSecs <= 0 {
		req.StartupWaitSecs = 8
	}
	return nil
}
```