# Scenario

**Feature**: keep-alive management HTTP restart endpoints

```
# server signal vs daemon exec — different POST paths, different side effects
test daemon -> GET /api/keep-alive/status -> POST restart endpoint -> settle -> status
```

## Preconditions

Session lock held (root `Setup`). Isolated `AI_CRITIC_HOME` with test credentials.

## Steps

1. Leaf `Setup` sets `Op` to `api-restart-server` or `api-restart-daemon`.

## Context

API-level contract for macOS client parity with web Manage Server **Restart Daemon**.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	if req.StartupWaitSecs <= 0 {
		req.StartupWaitSecs = 15
	}
	if req.SettleWaitSecs <= 0 {
		req.SettleWaitSecs = 20
	}
	return nil
}
```