## Expected

- `BuildSingBoxTunConfig` succeeds.
- VMess outbound `server` is `ws-golden.example.com`, `server_port` 8443.
- `transport.path` is `/tunnel/ws` (sing-box flat V2Ray transport, not nested `transport.ws`).
- `tls.enabled` is true; `tls.server_name` is `ws-golden.example.com`.

## Side Effects

- None (pure function).

## Errors

- None.

## Exit Code

- Success.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.RunErr != nil {
		t.Fatalf("build error: %v", resp.RunErr)
	}
	out := findOutbound(resp.ConfigJSON, "vmess")
	if out == nil {
		t.Fatalf("missing vmess outbound: %v", resp.ConfigJSON)
	}
	if out["server"] != "ws-golden.example.com" {
		t.Fatalf("server = %v", out["server"])
	}
	port := out["server_port"]
	if port != float64(8443) && port != int(8443) {
		t.Fatalf("server_port = %v, want 8443", port)
	}
	transport, _ := out["transport"].(map[string]any)
	if transport["type"] != "ws" {
		t.Fatalf("transport.type = %v, want ws", transport["type"])
	}
	if transport["path"] != "/tunnel/ws" {
		t.Fatalf("transport.path = %v", transport["path"])
	}
	tls, _ := out["tls"].(map[string]any)
	if tls["enabled"] != true {
		t.Fatalf("tls.enabled = %v", tls["enabled"])
	}
	if tls["server_name"] != "ws-golden.example.com" {
		t.Fatalf("tls.server_name = %v", tls["server_name"])
	}
}
```