## Expected

- VMess outbound `tls.enabled` is false when API returns `tls: "none"`.

## Side Effects

- None.

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
		t.Fatal("missing vmess outbound")
	}
	tls, ok := out["tls"].(map[string]any)
	if !ok {
		t.Fatal("tls block missing on outbound")
	}
	if tls["enabled"] != false {
		t.Fatalf("tls.enabled = %v, want false for tls:none", tls["enabled"])
	}
}
```