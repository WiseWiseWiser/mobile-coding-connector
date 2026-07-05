# Scenario

**Feature**: Swift menu-bar restart action contract

```
# menu button label and handler must target daemon exec restart
User -> Button("Restart Daemon") -> DaemonClient.restartDaemon() -> POST /restart-daemon
```

## Preconditions

Swift sources exist under `macos-ai-critic/ai-critic-macos/`.

## Steps

1. Set `Op=client`.

## Context

Pure source inspection — no subprocess or HTTP.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "client"
	return nil
}
```