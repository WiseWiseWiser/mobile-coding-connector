## Expected

1. Session.LocalPort == 18080.
2. Session.ProxyPort > 0 and != LocalPort.
3. TunnelStartPort == ProxyPort (provider started against hop, not app).

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.StartErr != "" {
		t.Fatalf("StartErr: %s", resp.StartErr)
	}
	if resp.Session == nil {
		t.Fatal("expected session")
	}
	if resp.Session.LocalPort != defaultTestPort {
		t.Fatalf("LocalPort=%d want %d", resp.Session.LocalPort, defaultTestPort)
	}
	if resp.Session.ProxyPort <= 0 {
		t.Fatal("ProxyPort must be set")
	}
	if resp.Session.ProxyPort == resp.Session.LocalPort {
		t.Fatalf("ProxyPort must differ from LocalPort; both %d", resp.Session.ProxyPort)
	}
	if resp.TunnelStartPort != resp.Session.ProxyPort {
		t.Fatalf("TunnelStartPort=%d want ProxyPort=%d (hop)", resp.TunnelStartPort, resp.Session.ProxyPort)
	}
}
```
