# Scenario

**Feature**: paste-bin CLI validation and auth failures

```
# bad args or token -> remote-agent paste-bin -> non-zero exit
remote-agent paste-bin (invalid) -> Error on stderr/combined
```

## Preconditions

Server may be running; failures occur before or during API calls.

## Steps

1. Configure invalid CLI invocation (extra args or bad token).
2. Assert non-zero exit and actionable error text.

## Context

Surface errors for unknown positional args and authorization failures.

```go
import (
	"testing"

	"github.com/xhd2015/ai-critic/script/lib"
)

func Setup(t *testing.T, req *Request) error {
	if req.Token == "" {
		req.Token = lib.TestPassword
	}
	return nil
}
```