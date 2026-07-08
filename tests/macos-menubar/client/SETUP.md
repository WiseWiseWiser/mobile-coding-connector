# Scenario

**Feature**: Swift AppState grok/codex refresh uses server port

```
AppState.refresh -> ServerClient.grokUsage/codexUsage on :23712 (not DaemonClient)
```

## Preconditions

Swift sources under `macos-ai-critic/ai-critic-macos/`.

## Steps

1. Set `Op=client`.

## Context

REQUIREMENT section E — menubar contract for server-port business APIs.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "client"
	return nil
}
```