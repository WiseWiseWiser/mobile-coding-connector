# Scenario

**Feature**: Swift menu-bar restart action contract

```
# menu button label and handler must target daemon exec restart
User -> Button("Restart Daemon") -> DaemonClient.restartDaemon() -> POST /restart-daemon
```

## Preconditions

Swift sources exist under `macos-ai-critic/ai-critic-macos/`.

## Steps

1. Leaf sets `Op` (`client-restart` or `client-business-port`).

## Context

Pure source inspection — no subprocess or HTTP. Leaf `SETUP.md` files assign
`req.Op` (`client-restart` or `client-business-port`) after this grouping node.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	// Grouping node only — descendant leaf Setup sets req.Op before Run.
	if req.SettleWaitSecs <= 0 {
		req.SettleWaitSecs = 20
	}
	return nil
}
```