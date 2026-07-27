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
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "client"
	return nil
}
```