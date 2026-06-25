## Preconditions

Tests run against the `wsproxy` package with an isolated config directory and
optional `httptest` server simulating xray's WebSocket endpoint (GET `/ws` → 400).

## Steps

1. Child `Setup` configures `Request` fields for the scenario.
2. Root `Run` writes `ws-proxy.json`, optionally starts the fake xray server.
3. `Run` constructs a test `Manager` and collects status, tunnel mapping, VMess link.

## Context

These tests encode the reported production bug: `remote-agent ws-proxy status`
shows `Running: true` and a `Public URL`, but V2Ray clients fail with
`ERR_PROXY_CONNECTION_FAILED` because Cloudflare returns HTTP 404 (no ingress)
while only local xray health is checked.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	return nil
}
```