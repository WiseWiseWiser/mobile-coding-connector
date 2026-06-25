## Expected

The vmess-link endpoint must refuse to serve a link when tunnel ingress is not
registered, even if local xray is alive.

1. `Response.LocalXrayAlive` is `true`.
2. `Response.TunnelMappingPresent` is `false`.
3. `Response.VMessLink` is empty — API should return not-ready instead of a broken link.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}
	if !resp.LocalXrayAlive {
		t.Fatal("precondition failed: xray should be alive")
	}
	if resp.TunnelMappingPresent {
		t.Fatal("precondition failed: tunnel mapping must be absent")
	}
	if resp.VMessLink != "" {
		t.Fatalf("vmess link must be withheld without tunnel ingress; got %q", resp.VMessLink)
	}
}
```