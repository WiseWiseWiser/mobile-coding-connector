## Expected

After restart with orphan xray:

1. `Response.LocalXrayAlive` is `true`.
2. `Response.StatusPublicURL` is restored as `https://ws-25b2a55939e4.xhd2015.xyz` from persisted `ws-proxy.json`.
3. `Response.ClientReady` is `false` until tunnel ingress is restored.
4. `Response.VMessLink` is empty without tunnel ingress.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}
	if !resp.LocalXrayAlive {
		t.Fatal("precondition failed: orphan xray should still respond on /ws")
	}
	wantURL := "https://ws-25b2a55939e4.xhd2015.xyz"
	if resp.StatusPublicURL != wantURL {
		t.Fatalf("Status.PublicURL = %q, want %q (restored from ws-proxy.json after restart)", resp.StatusPublicURL, wantURL)
	}
	if resp.ClientReady {
		t.Fatal("client-ready must be false without tunnel ingress after restart")
	}
	if resp.VMessLink != "" {
		t.Fatalf("vmess link must be withheld without tunnel ingress; got %q", resp.VMessLink)
	}
}
```