## Expected

- `Run` succeeds (`RunErr` is nil).
- `FetchVMessCalled` is true.
- Stdout is valid JSON with `tun` inbound and `vmess` outbound.
- Outbound reflects mock host `ws-test.example.com` and path `/ws`.

## Side Effects

- No output file written.

## Errors

- None.

## Exit Code

- Success (implicit via nil `RunErr`).

```go
import (
	"encoding/json"
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.RunErr != nil {
		t.Fatalf("client-config error: %v", resp.RunErr)
	}
	if !resp.FetchVMessCalled {
		t.Fatal("FetchVMess must be called")
	}
	var cfg map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(resp.Stdout)), &cfg); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\nstdout=%q", err, resp.Stdout)
	}
	if !configHasTunInbound(cfg) {
		t.Fatalf("missing tun inbound: %v", cfg["inbounds"])
	}
	out := findOutbound(cfg, "vmess")
	if out == nil {
		t.Fatalf("missing vmess outbound: %v", cfg["outbounds"])
	}
	server, _ := out["server"].(string)
	if server != "ws-test.example.com" {
		t.Fatalf("server = %q, want ws-test.example.com", server)
	}
	transport, _ := out["transport"].(map[string]any)
	if transport["path"] != "/ws" {
		t.Fatalf("transport.path = %v, want /ws", transport["path"])
	}
}
```