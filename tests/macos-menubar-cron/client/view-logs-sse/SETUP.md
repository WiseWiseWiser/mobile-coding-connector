# Scenario

**Feature**: View Logs streams via server SSE on both apps

```
View Logs -> GET /api/logs/stream?path=...&lines=... (SSE; not local tail)
```

## Preconditions

Swift sources for local and/or remote menu-bar apps are present.

## Steps

1. Set `ClientLeaf=view-logs-sse`.

## Context

REQUIREMENT leaf: `client/view-logs-sse`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.ClientLeaf = "view-logs-sse"
	return nil
}
```
