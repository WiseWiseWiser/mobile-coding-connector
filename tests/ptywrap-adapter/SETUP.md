# Scenario

**Feature**: ai-critic terminal routes delegate to ptywrap adapter

```
# adapter regression
ai-critic-server -> ptywrap adapter -> shared ptywrap library
```

## Preconditions

- `ai-critic-terminal` module builds (`go build .` from module root).
- Terminal routes registered after refactor via thin adapter.

## Steps

1. Build and start `ai-critic-server` on ephemeral port with test credentials.
2. Set `req.ServerURL` for descendant leaves.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	base, port, cleanup := startAICriticServer(t)
	t.Cleanup(cleanup)
	req.ServerURL = base
	req.ServerPort = port
	return nil
}
```