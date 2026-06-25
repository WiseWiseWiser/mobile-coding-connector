## Expected

When local xray is healthy but the Cloudflare ingress mapping is missing:

1. `Response.LocalXrayAlive` is `true`.
2. `Response.TunnelMappingPresent` is `false`.
3. `Response.ClientReady` is `false` — VMess clients cannot connect.
4. `Response.StatusRunning` is `false` — status must not claim the proxy is running without tunnel ingress.

## Errors

- Status reports `Running: true` while tunnel mapping is absent (current bug).
- VMess link is issued despite unreachable public hostname.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}
	if !resp.LocalXrayAlive {
		t.Fatal("precondition failed: simulated xray should be alive on /ws")
	}
	if resp.TunnelMappingPresent {
		t.Fatal("precondition failed: tunnel mapping must be absent for this repro")
	}
	if resp.VMessLink != "" {
		t.Fatal("precondition failed: vmess link should be withheld without tunnel mapping")
	}
	if resp.StatusRunning {
		t.Fatalf("BUG: Status.Running=true with publicURL=%q but tunnel mapping missing; clients see ERR_PROXY_CONNECTION_FAILED (Cloudflare 404)", resp.StatusPublicURL)
	}
	if resp.ClientReady {
		t.Fatalf("client-ready must be false without tunnel mapping; got ClientReady=true publicURL=%q", resp.StatusPublicURL)
	}
}
```