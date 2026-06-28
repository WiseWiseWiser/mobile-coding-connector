# Scenario

**Feature**: openclaw mock integration doctest harness

```
# isolated data dir per leaf; Manager reads/writes config and state files
doctest Run -> SetTestDataDir -> Manager/API -> Response

# secrets plaintext on disk, masked on API GET/PUT only
Config store -> MaskConfig -> API response
```

## Preconditions

- `openclaw.SetTestDataDir` redirects all config/state paths for isolation.
- Mock mode: no real `openclaw gateway` subprocess or Slack WebSocket.
- `GetManager()` singleton is safe because state lives on disk per test dir.

## Steps

1. Child `Setup` sets `Request.Op` and scenario-specific fields.
2. Root `Run` creates `t.TempDir()`, applies config preconditions, executes operation.
3. Leaf `Assert` validates `Response` against expected outcomes.

## Context

Extends unit test coverage with doctor, dry-run, HTTP status codes, stop
idempotency, partial PUT without slack block, and generated-config field assertions.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	if req.GatewayPort == 0 {
		req.GatewayPort = 18789
	}
	return nil
}
```