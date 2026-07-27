# Scenario

**Feature**: business APIs use server port 23712

```
ServerClient -> :23712 /api/grok/usage, /api/codex/usage, /api/services?all=1
```

## Preconditions

Daemon port `23312` is control-plane only; business APIs moved to server.

## Steps

1. Set `ClientLeaf=swift-server-port`.

## Context

REQUIREMENT leaf: `client/swift-server-port`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.ClientLeaf = "swift-server-port"
	return nil
}
```